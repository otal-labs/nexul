import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

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

export const useMarkNotificationRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post(`/api/notifications/${id}/read`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
  });
};

export const useMarkAllNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () =>
      api.post(inWorkspace("/api/notifications/read-all", useWorkspaceStore.getState().selectedWorkspaceId)),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
  });
};

// Hides the tab badge once nothing is unread, rather than showing a stray "0".
export const unreadBadge = (count: number | undefined): number | undefined =>
  count && count > 0 ? count : undefined;
