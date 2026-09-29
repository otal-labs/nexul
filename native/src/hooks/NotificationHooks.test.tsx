import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { unreadBadge, useFetchNotifications, useFetchUnreadCount } from "@/hooks/NotificationHooks";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
}));

const notification = {
  id: "n1",
  user_id: "u1",
  kind: "doc.created",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: false,
  created_at: "2026-08-12T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.post).mockReset();
});

describe("useFetchNotifications", () => {
  test("loads the notification list", async () => {
    jest.mocked(api.get).mockResolvedValue([notification]);
    const { result } = await renderHook(() => useFetchNotifications(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([notification]));
    expect(api.get).toHaveBeenCalledWith("/api/notifications");
  });
});

describe("useFetchUnreadCount", () => {
  test("loads the unread count", async () => {
    jest.mocked(api.get).mockResolvedValue({ count: 2 });
    const { result } = await renderHook(() => useFetchUnreadCount(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ count: 2 }));
    expect(api.get).toHaveBeenCalledWith("/api/notifications/unread-count");
  });
});

describe("unreadBadge", () => {
  test("hides the badge when there is nothing unread", () => {
    expect(unreadBadge(0)).toBeUndefined();
    expect(unreadBadge(undefined)).toBeUndefined();
  });

  test("shows the count once something is unread", () => {
    expect(unreadBadge(3)).toBe(3);
  });
});
