import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Notification, UnreadCount } from "@/models/Notification";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getNotificationsKey = "getNotifications";
export const getUnreadCountKey = "getUnreadCount";

// The inbox and its badge follow the selected workspace, like every other workspace-scoped page.
export const useFetchNotifications = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getNotificationsKey, workspaceId],
    queryFn: async () =>
      (await api.get<Notification[]>("/api/notifications", { params: { workspace_id: workspaceId } })).data,
  });
};

export const useFetchUnreadCount = (enabled = true) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getUnreadCountKey, workspaceId],
    queryFn: async () =>
      (await api.get<UnreadCount>("/api/notifications/unread-count", { params: { workspace_id: workspaceId } }))
        .data,
    enabled,
  });
};

export const useMarkNotificationRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.post(`/api/notifications/${id}/read`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMarkAllNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () =>
      api.post("/api/notifications/read-all", undefined, {
        params: { workspace_id: useWorkspaceStore.getState().selectedWorkspaceId },
      }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
      toast.success("All notifications marked read");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
