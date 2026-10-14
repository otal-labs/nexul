import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { defineQuery } from "@/lib/liveQuery";
import type { Notification, UnreadCount } from "@/models/Notification";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getNotificationsKey = "getNotifications";
export const getUnreadCountKey = "getUnreadCount";

const inWorkspace = (path: string, workspaceId: string | undefined) =>
  `${path}?workspace_id=${encodeURIComponent(workspaceId ?? "")}`;

const notificationsQuery = defineQuery({
  key: getNotificationsKey,
  fetch: (workspaceId: string | undefined) => api.get<Notification[]>(inWorkspace("/api/notifications", workspaceId)),
  refreshes: { "notification.created": "all" },
});

const unreadCountQuery = defineQuery({
  key: getUnreadCountKey,
  fetch: (workspaceId: string | undefined) => api.get<UnreadCount>(inWorkspace("/api/notifications/unread-count", workspaceId)),
  refreshes: { "notification.created": "all" },
});

// The Inbox and its tab badge follow the selected workspace, like every other tab.
export const useFetchNotifications = () => {
  const workspaceId = useCurrentWorkspaceId();
  return useQuery({ ...notificationsQuery.options(workspaceId), enabled: !!workspaceId });
};

export const useFetchUnreadCount = (enabled = true) => {
  const workspaceId = useCurrentWorkspaceId();
  return useQuery({ ...unreadCountQuery.options(workspaceId), enabled: enabled && !!workspaceId });
};

const refreshNotifications = async (client: QueryClient) => {
  await client.invalidateQueries({ queryKey: [getNotificationsKey] });
  await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
};

export const useMarkNotificationRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", (id: string) => api.post(`/api/notifications/${id}/read`)),
    onSuccess: () => refreshNotifications(client),
  });
};

export const useMarkNotificationUnread = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", (id: string) => api.post(`/api/notifications/${id}/unread`)),
    onSuccess: () => refreshNotifications(client),
  });
};

export const useMarkAllNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", () =>
      api.post(`/api/notifications/read-all?workspace_id=${encodeURIComponent(useWorkspaceStore.getState().selectedWorkspaceId ?? "")}`)),
    onSuccess: () => refreshNotifications(client),
  });
};

// Hides the tab badge once nothing is unread, rather than showing a stray "0".
export const unreadBadge = (count: number | undefined): number | undefined =>
  count && count > 0 ? count : undefined;
