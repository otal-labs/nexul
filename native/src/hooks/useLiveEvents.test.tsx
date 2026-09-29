import { QueryClient } from "@tanstack/react-query";

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
});
