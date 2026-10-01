import { QueryClient } from "@tanstack/react-query";

import { getMeKey } from "@/hooks/AuthHooks";
import { getChatConversationsKey, getChatUnreadKey } from "@/hooks/ChatHooks";
import { getNotificationsKey, getUnreadCountKey } from "@/hooks/NotificationHooks";
import { dispatch } from "@/hooks/useLiveEvents";

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
});
