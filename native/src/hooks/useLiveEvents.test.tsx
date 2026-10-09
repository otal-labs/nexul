import { QueryClient } from "@tanstack/react-query";
import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

import { getMeKey } from "@/hooks/AuthHooks";
import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDocKey } from "@/hooks/DocHooks";
import { getTicketKey } from "@/hooks/TicketHooks";
import { getChatConversationsKey, getChatMessagesKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { dispatch } from "@/hooks/useLiveEvents";
import { getMyRoleKey, getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import type { Message } from "@/models/Chat";

jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));

describe("dispatch", () => {
  test("notification.created invalidates the notification list and the unread count", () => {
    const client = new QueryClient();
    const invalidate = jest.spyOn(client, "invalidateQueries").mockResolvedValue();

    dispatch(client)({ topic: "notification.created", type: "event", payload: {} });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getNotificationsKey] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getUnreadCountKey] });
  });

  test("a channel's members changing refetches the conversation list and the unread counts", () => {
    const client = new QueryClient();
    const invalidate = jest.spyOn(client, "invalidateQueries").mockResolvedValue();

    dispatch(client)({ topic: "chat.conversation.members_changed", type: "event", payload: { conversation_id: "c1" } });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getChatConversationsKey] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getChatUnreadKey] });
  });

  test.each(["access.grant.changed", "workspace.member.updated"])(
    "%s naming the viewer refetches every open read, and naming someone else refetches none",
    (topic) => {
      const client = new QueryClient();
      client.setQueryData([getMeKey], { user: { id: "u1", login: "onik", name: "Onik" } });
      const invalidate = jest.spyOn(client, "invalidateQueries").mockResolvedValue();

      dispatch(client)({ topic, type: "event", payload: { resource_type: "project", resource_id: "p1", user_id: "u2" } });
      expect(invalidate).not.toHaveBeenCalledWith();

      dispatch(client)({ topic, type: "event", payload: { resource_type: "project", resource_id: "p1", user_id: "u1" } });
      expect(invalidate).toHaveBeenCalledWith();
    },
  );

  // A frame names the change, so an open thread updates without downloading its page of messages again.
  const message = (id: string, body: string): Message => ({
    id,
    conversation_id: "c1",
    author_id: "u2",
    author_kind: "user",
    body,
    mentions: null,
    created_at: "2026-10-09T10:00:00Z",
    updated_at: "2026-10-09T10:00:00Z",
  });
  const first = { ...message("m1", "hello"), reactions: [{ emoji: "👍", user_ids: ["u1"] }] };

  test.each([
    ["chat.message.created", { message: message("m2", "new") }, [first, message("m2", "new")]],
    ["chat.message.updated", { message: { ...first, body: "edited" } }, [{ ...first, body: "edited" }]],
    [
      "chat.message.deleted",
      { conversation_id: "c1", message_id: "m1", deleted_at: "2026-10-09T11:00:00Z" },
      [{ ...first, deleted_at: "2026-10-09T11:00:00Z" }],
    ],
    [
      "chat.message.reactions_changed",
      { conversation_id: "c1", message_id: "m1", user_id: "u2", emoji: "👍", reacted: true },
      [{ ...first, reactions: [{ emoji: "👍", user_ids: ["u1", "u2"] }] }],
    ],
  ])("%s patches the cached thread instead of refetching it", (topic, payload, want) => {
    const client = new QueryClient();
    client.setQueryData([getChatMessagesKey, "c1"], [first]);
    const invalidate = jest.spyOn(client, "invalidateQueries").mockResolvedValue();

    dispatch(client)({ topic, type: "event", payload });

    expect(client.getQueryData([getChatMessagesKey, "c1"])).toEqual(want);
    expect(invalidate.mock.calls.flatMap(([filters]) => filters?.queryKey ?? [])).not.toContain(getChatMessagesKey);
  });

  // Each record opened or listed caches its own detail query, so a frame refetches only the one it names.
  test.each([
    ["ticket.status_changed", { ticket: { id: "rec-1" }, from: "a", to: "b" }, [getTicketKey]],
    ["ticket.updated", { ticket: { id: "rec-1" } }, [getTicketKey]],
    ["ticket.deleted", { id: "rec-1", title: "Gone" }, [getTicketKey]],
    ["doc.updated", { doc: { id: "rec-1", title: "Spec", version: 2 } }, [getDocKey]],
    ["deploy.updated", { id: "rec-1", status: "running" }, [getDeployKey, getDeployLogKey]],
  ])("%s refetches only the record it names", (topic, payload, keys) => {
    const client = new QueryClient();
    keys.forEach((key) => {
      client.setQueryData([key, "rec-1"], { id: "rec-1" });
      client.setQueryData([key, "rec-2"], { id: "rec-2" });
    });

    dispatch(client)({ topic, type: "event", payload });

    keys.forEach((key) => {
      expect(client.getQueryState([key, "rec-1"])?.isInvalidated).toBe(true);
      expect(client.getQueryState([key, "rec-2"])?.isInvalidated).toBe(false);
    });
  });

  test("a ticket frame reaches a ticket a chat link opened by its key", () => {
    const client = new QueryClient();
    client.setQueryData([getTicketKey, "WEB-12", "acme"], { id: "t-1" });
    client.setQueryData([getTicketKey, "WEB-13", "acme"], { id: "t-2" });

    dispatch(client)({ topic: "ticket.updated", type: "event", payload: { ticket: { id: "t-1" } } });

    expect(client.getQueryState([getTicketKey, "WEB-12", "acme"])?.isInvalidated).toBe(true);
    expect(client.getQueryState([getTicketKey, "WEB-13", "acme"])?.isInvalidated).toBe(false);
  });

  // A query cached with an Infinity stale time only refreshes when a pushed topic invalidates it.
  const referenceKeys: [string, string, unknown][] = [
    [getMeKey, "account.profile_updated", { account_id: "u2" }],
    [getWorkspacePeopleKey, "account.profile_updated", { account_id: "u2" }],
    [getWorkspacesKey, "workspace.updated", { workspace_id: "ws-1", name: "Acme", slug: "acme" }],
    [getMyRoleKey, "role.updated", { role_id: "r1", workspace_id: "ws-1" }],
  ];

  test.each(referenceKeys)("%s never goes stale on its own, so %s refreshes it", (key, topic, payload) => {
    const client = new QueryClient();
    client.setQueryData([key, "ws-1"], {});

    dispatch(client)({ topic, type: "event", payload });

    expect(client.getQueryState([key, "ws-1"])?.isInvalidated).toBe(true);
  });

  // Read from the source, so a new query cached forever fails here until it gets a row and a topic above.
  const infinityQueryKeys = () => {
    const root = join(__dirname, "..");
    const sources = readdirSync(root, { recursive: true, encoding: "utf8" })
      .filter((file) => /\.tsx?$/.test(file) && !file.includes(".test."))
      .map((file) => readFileSync(join(root, file), "utf8"));
    const values = new Map(sources.flatMap((source) => [...source.matchAll(/export const (\w+) = "([^"]+)"/g)].map(([, name, value]) => [name, value])));
    return sources.flatMap((source) =>
      source
        .split(/\buseQuery\(|\bqueryOptions\(/)
        .slice(1)
        .filter((query) => /referenceDataOptions|staleTime: Infinity/.test(query))
        .map((query) => values.get(/queryKey: \[(\w+)/.exec(query)?.[1] ?? "")),
    );
  };

  test("every query cached forever has a row in the table above", () => {
    expect(new Set(infinityQueryKeys())).toEqual(new Set(referenceKeys.map(([key]) => key)));
  });
});
