import { QueryClient } from "@tanstack/react-query";

import { getMeKey } from "@/hooks/AuthHooks";
import { getChatConversationsKey, getChatMessagesKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { dispatch } from "@/hooks/useLiveEvents";
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
});
