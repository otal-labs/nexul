import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useFetchInbox,
  useFetchUnreadCount,
  useMarkAllNotificationsRead,
  useMarkNotificationsRead,
  notificationFollower,
} from "@/hooks/NotificationHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Notification } from "@/models/Notification";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("useFetchInbox", () => {
  it("loads the selected workspace's notifications", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [notification] });
    const { result } = renderHook(() => useFetchInbox(), { wrapper });
    await waitFor(() => expect(result.current.data?.map((e) => e.key)).toEqual(["doc:doc-1"]));
    expect(api.get).toHaveBeenCalledWith("/api/notifications", { params: { workspace_id: "ws-1" } });
  });
});

describe("useFetchUnreadCount", () => {
  it("counts only the selected workspace's unread notifications", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { count: 2 } });
    const { result } = renderHook(() => useFetchUnreadCount(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ count: 2 }));
    expect(api.get).toHaveBeenCalledWith("/api/notifications/unread-count", { params: { workspace_id: "ws-1" } });
  });
});

describe("useMarkNotificationsRead", () => {
  it("posts the read endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useMarkNotificationsRead(), { wrapper });
    await result.current.mutateAsync(["n1"]);
    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useMarkNotificationsRead(), { wrapper });
    await result.current.mutateAsync(["n1"]).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useMarkAllNotificationsRead", () => {
  it("marks only the selected workspace read", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useMarkAllNotificationsRead(), { wrapper });
    await result.current.mutateAsync();
    expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all", undefined, {
      params: { workspace_id: "ws-1" },
    });
  });
});

describe("the notification follower", () => {
  const row = (id: string, subject: string, folder: string) =>
    ({
    ...notification,
    id,
    subject_type: "doc",
    subject_id: subject,
    folder_id: folder,
    folder_name: "Specs",
    folder_is_default: false,
    }) as Notification;
  const inboxes = () =>
    seeded([
      [["getNotifications", "ws-1"], [row("n-1", "d-1", "f-1")]],
      [["getNotifications", "ws-2"], [row("n-2", "d-2", "f-2")]],
    ]);

  it("refetches only the inbox with a row about a moved doc", async () => {
    const client = inboxes();
    await followFrame(notificationFollower, "doc.moved", { doc: { id: "d-1" }, from_folder_id: "f-1" }, client);
    expect([isStale(client, ["getNotifications", "ws-1"]), isStale(client, ["getNotifications", "ws-2"])]).toEqual([true, false]);
  });

  it("renames a folder on the inbox rows grouped under it without a request", async () => {
    const client = inboxes();
    await followFrame(notificationFollower, "doc.folder.updated", { folder: { id: "f-2", project_id: "p-1", name: "Plans", is_default: false } }, client);
    expect(client.getQueryData<Notification[]>(["getNotifications", "ws-2"])?.[0]?.folder_name).toBe("Plans");
    expect(isStale(client, ["getNotifications", "ws-2"])).toBe(false);
  });

  it("refetches every inbox and badge on a new notice, which names no workspace", async () => {
    const client = inboxes();
    client.setQueryData(["getUnreadCount"], { count: 0, workspaces: {} });
    await followFrame(notificationFollower, "notification.created", {}, client);
    expect([["getNotifications", "ws-1"], ["getNotifications", "ws-2"], ["getUnreadCount"]].map((key) => isStale(client, key))).toEqual([true, true, true]);
  });
});
