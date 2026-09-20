import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useFetchNotifications,
  useFetchUnreadCount,
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from "@/hooks/NotificationHooks";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

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
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("useFetchNotifications", () => {
  it("loads the notification list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [notification] });
    const { result } = renderHook(() => useFetchNotifications(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([notification]));
    expect(api.get).toHaveBeenCalledWith("/api/notifications");
  });
});

describe("useFetchUnreadCount", () => {
  it("loads the unread count", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { count: 2 } });
    const { result } = renderHook(() => useFetchUnreadCount(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ count: 2 }));
    expect(api.get).toHaveBeenCalledWith("/api/notifications/unread-count");
  });
});

describe("useMarkNotificationRead", () => {
  it("posts the read endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useMarkNotificationRead(), { wrapper });
    await result.current.mutateAsync("n1");
    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useMarkNotificationRead(), { wrapper });
    await result.current.mutateAsync("n1").catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useMarkAllNotificationsRead", () => {
  it("posts the read-all endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useMarkAllNotificationsRead(), { wrapper });
    await result.current.mutateAsync();
    expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all");
  });
});
