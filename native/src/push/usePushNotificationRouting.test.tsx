import { renderHook, waitFor } from "@testing-library/react-native";
import * as Notifications from "expo-notifications";

import { api } from "@/api/client";
import { getNotificationsKey } from "@/hooks/NotificationHooks";
import { queryClient } from "@/lib/queryClient";
import type { Notification } from "@/models/Notification";
import { usePushNotificationRouting } from "@/push/usePushNotificationRouting";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

let responseListener: ((response: unknown) => void) | undefined;
jest.mock("expo-notifications", () => ({
  getLastNotificationResponse: jest.fn(() => null),
  clearLastNotificationResponse: jest.fn(),
  addNotificationResponseReceivedListener: jest.fn((listener: (response: unknown) => void) => {
    responseListener = listener;
    return { remove: jest.fn() };
  }),
}));

jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const notification: Notification = {
  id: "n1",
  user_id: "u1",
  workspace_id: "ws-1",
  kind: "ticket.assigned",
  subject_type: "ticket",
  subject_id: "t1",
  subject_title: "Fix login",
  read: false,
  created_at: "2026-01-01T00:00:00.000Z",
};

const respond = (data: { notification_id: string }) => ({
  notification: { request: { content: { data } } },
});

describe("usePushNotificationRouting", () => {
  beforeEach(() => {
    mockPush.mockReset();
    queryClient.clear();
    responseListener = undefined;
    jest.mocked(api.get).mockReset();
    jest.mocked(Notifications.getLastNotificationResponse).mockReturnValue(null);
  });

  test("a warm tap finds the notification the Inbox already holds, opens the Inbox, then its subject", async () => {
    queryClient.setQueryData([getNotificationsKey, "ws-1"], [notification]);
    await renderHook(() => usePushNotificationRouting());

    responseListener?.(respond({ notification_id: "n1" }));

    await waitFor(() => expect(mockPush).toHaveBeenNthCalledWith(2, "/board/ticket/t1", { withAnchor: true }));
    expect(mockPush).toHaveBeenNthCalledWith(1, "/inbox");
    expect(api.get).not.toHaveBeenCalled();
  });

  test("a cold start with a pending response asks the server and routes the same way", async () => {
    jest.mocked(api.get).mockResolvedValue([notification]);
    jest.mocked(Notifications.getLastNotificationResponse).mockReturnValue(respond({ notification_id: "n1" }) as never);

    await renderHook(() => usePushNotificationRouting());

    await waitFor(() => expect(mockPush).toHaveBeenNthCalledWith(2, "/board/ticket/t1", { withAnchor: true }));
    expect(Notifications.clearLastNotificationResponse).toHaveBeenCalled();
  });

  test("an unknown notification id still opens the Inbox and stops there", async () => {
    queryClient.setQueryData([getNotificationsKey, "ws-1"], [notification]);
    jest.mocked(api.get).mockResolvedValue([notification]);
    await renderHook(() => usePushNotificationRouting());

    responseListener?.(respond({ notification_id: "missing" }));

    await waitFor(() => expect(mockPush).toHaveBeenCalledWith("/inbox"));
    expect(mockPush).toHaveBeenCalledTimes(1);
  });
});
