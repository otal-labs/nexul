import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { unreadBadge, useFetchNotifications, useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), post: jest.fn() },
}));

const notification = {
  id: "n1",
  user_id: "u1",
  workspace_id: "ws-1",
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

const serve = (path: string, body: unknown) =>
  jest.mocked(api.get).mockImplementation((requested: string) => {
    if (requested === "/api/workspaces") return Promise.resolve([{ id: "ws-1", name: "Acme" }]);
    if (requested === path) return Promise.resolve(body);
    return Promise.reject(new Error(`unexpected GET ${requested}`));
  });

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.post).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("useFetchNotifications", () => {
  test("loads only the selected workspace's notifications", async () => {
    serve("/api/notifications?workspace_id=ws-1", [notification]);
    const { result } = await renderHook(() => useFetchNotifications(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([notification]));
  });
});

describe("useFetchUnreadCount", () => {
  test("counts only the selected workspace's unread notifications, for the tab badge", async () => {
    serve("/api/notifications/unread-count?workspace_id=ws-1", { count: 2 });
    const { result } = await renderHook(() => useFetchUnreadCount(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ count: 2 }));
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
