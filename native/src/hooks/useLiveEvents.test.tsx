import { eventFixtures, TOPICS } from "@nexul/sdk/events";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react-native";
import { readdirSync } from "node:fs";
import { join } from "node:path";
import type { ReactNode } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { getMeKey } from "@/hooks/AuthHooks";
import { getChatConversationsKey, getChatMessagesKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getDeployKey, getDeployLogKey } from "@/hooks/DeployHooks";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getProjectStatusesKey } from "@/hooks/StatusHooks";
import { getTicketKey, getTicketsByProjectKey } from "@/hooks/TicketHooks";
import { dispatch, useLiveEvents } from "@/hooks/useLiveEvents";
import { getMyRoleKey, getWorkspacesKey } from "@/hooks/WorkspaceHooks";
import { liveQueries } from "@/lib/liveQuery";
import type { Message } from "@/models/Chat";
import { useSessionStore } from "@/stores/sessionStore";

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

  // Each frame finds its entry from the payload, so a frame about one workspace, project or thread leaves the others alone.
  test.each([
    ["chat.conversation.deleted", { conversation_id: "c1", workspace_id: "ws-1", kind: "channel", name: "x" }, getChatMessagesKey, "c1", "c2"],
    ["chat.conversation.members_changed", { conversation_id: "c1", workspace_id: "ws-1" }, getChatUnreadKey, "ws-1", "ws-2"],
    ["role.updated", { role_id: "r1", workspace_id: "ws-1" }, getMyRoleKey, "ws-1", "ws-2"],
    ["workspace.member.removed", { user_id: "u2", workspace_id: "ws-1" }, getWorkspacePeopleKey, "ws-1", "ws-2"],
    ["ticket.status_changed", { ticket: { id: "t-1", project_id: "p-1" }, from: "a", to: "b" }, getTicketsByProjectKey, "p-1", "p-2"],
    ["status.updated", { status: { id: "st-1", project_id: "p-1" } }, getProjectStatusesKey, "p-1", "p-2"],
    ["doc.created", { doc: { id: "d1", project_id: "p-1", title: "Spec", version: 1 } }, getDocsKey, "p-1", "p-2"],
  ])("%s refetches only the entry its payload names", (topic, payload, key, named, other) => {
    const client = new QueryClient();
    client.setQueryData([key, named], []);
    client.setQueryData([key, other], []);

    dispatch(client)({ topic, type: "event", payload });

    expect(client.getQueryState([key, named])?.isInvalidated).toBe(true);
    expect(client.getQueryState([key, other])?.isInvalidated).toBe(false);
  });

  test.each([
    ["a channel refetches only its workspace's list", "channel", false],
    ["a DM refetches every workspace's list, since it shows wherever all its people belong", "dm", true],
  ])("a new conversation: %s", (_, kind, everyList) => {
    const client = new QueryClient();
    client.setQueryData([getChatConversationsKey, "ws-1"], []);
    client.setQueryData([getChatConversationsKey, "ws-2"], []);

    dispatch(client)({ topic: "chat.conversation.created", type: "event", payload: { conversation: { id: "c1", workspace_id: "ws-1", kind } } });

    expect(client.getQueryState([getChatConversationsKey, "ws-1"])?.isInvalidated).toBe(true);
    expect(client.getQueryState([getChatConversationsKey, "ws-2"])?.isInvalidated).toBe(everyList);
  });

  test("a frame that names no record refetches every cached one, rather than none", () => {
    const client = new QueryClient();
    client.setQueryData([getTicketKey, "t-1", ""], { id: "t-1" });
    client.setQueryData([getTicketKey, "t-2", ""], { id: "t-2" });

    dispatch(client)({ topic: "ticket.updated", type: "event", payload: { ticket: {} } });

    expect(client.getQueryState([getTicketKey, "t-1", ""])?.isInvalidated).toBe(true);
    expect(client.getQueryState([getTicketKey, "t-2", ""])?.isInvalidated).toBe(true);
  });
});

class FakeSocket {
  static all: FakeSocket[] = [];
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  readyState = 0;
  close = jest.fn();

  constructor(readonly url: string) {
    FakeSocket.all.push(this);
  }

  open() {
    this.readyState = 1;
    this.onopen?.({});
  }
}

describe("useLiveEvents", () => {
  let appStateListeners: ((status: AppStateStatus) => void)[] = [];
  const appStateChange = (status: AppStateStatus) => appStateListeners.forEach((listener) => listener(status));

  beforeEach(() => {
    jest.useFakeTimers();
    jest.spyOn(Math, "random").mockReturnValue(0);
    Object.assign(AppState, { currentState: "active" });
    appStateListeners = [];
    jest.spyOn(AppState, "addEventListener").mockImplementation((_, listener) => {
      appStateListeners.push(listener as (status: AppStateStatus) => void);
      return { remove: jest.fn() };
    });
    FakeSocket.all = [];
    global.WebSocket = FakeSocket as unknown as typeof WebSocket;
    useSessionStore.getState().signIn("https://nexul.example.com", "ses_a");
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  const mount = async () => {
    const client = new QueryClient();
    const invalidate = jest.spyOn(client, "invalidateQueries").mockResolvedValue();
    await renderHook(() => useLiveEvents(), {
      wrapper: ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    return invalidate;
  };

  test("a dropped socket reconnects and refetches every open read, since the frames sent meanwhile are lost", async () => {
    const invalidate = await mount();
    FakeSocket.all[0]?.open();
    expect(invalidate).not.toHaveBeenCalled();

    jest.spyOn(console, "warn").mockImplementation(() => undefined);
    await act(async () => {
      FakeSocket.all[0]?.onclose?.({});
      jest.advanceTimersByTime(500);
    });
    FakeSocket.all[1]?.open();

    expect(FakeSocket.all[1]?.url).toBe("wss://nexul.example.com/ws/events?token=ses_a");
    expect(invalidate).toHaveBeenCalledWith();
  });

  test("the background closes the socket and the foreground opens a new one, leaving the refetch to the focus manager", async () => {
    const invalidate = await mount();
    FakeSocket.all[0]?.open();

    await act(async () => appStateChange("background"));
    expect(FakeSocket.all[0]?.close).toHaveBeenCalled();
    await act(async () => appStateChange("active"));
    FakeSocket.all[1]?.open();

    expect(FakeSocket.all).toHaveLength(2);
    expect(invalidate).not.toHaveBeenCalled();
  });
});

// Every definition registers as its module loads, so loading every hooks module gives the whole declaration table.
readdirSync(__dirname)
  .filter((file) => /Hooks\.tsx$/.test(file))
  .forEach((file) => jest.requireActual(join(__dirname, file)));

// Derived from the definitions: a catalog topic touches the cache of exactly the queries that declare it, and no other.
test.each(TOPICS)("%s reaches exactly the queries that declare it", (topic) => {
  const client = new QueryClient();
  const reached = new Set<unknown>();
  jest.spyOn(client, "invalidateQueries").mockImplementation(async (filters) => void reached.add(filters?.queryKey?.[0]));
  jest.spyOn(client, "setQueriesData").mockImplementation((filters) => {
    reached.add(filters.queryKey?.[0]);
    return [];
  });

  dispatch(client)({ topic, type: "event", payload: eventFixtures[topic] });

  const declared = [...liveQueries()].filter((query) => topic in query.refreshes).map((query) => query.key);
  expect(reached).toEqual(new Set(declared));
});
