import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Notification, UnreadCount } from "@/models/Notification";
import { groupInbox, inboxRows, type InboxRow } from "@/utils/InboxUtility";

export const getNotificationsKey = "getNotifications";
export const getUnreadCountKey = "getUnreadCount";

// The inbox and its badge follow the selected workspace, like every other workspace-scoped page; the list comes grouped.
export const useFetchInbox = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getNotificationsKey, workspaceId],
    queryFn: async () =>
      (await api.get<Notification[]>("/api/notifications", { params: { workspace_id: workspaceId } })).data,
    select: groupInbox,
  });
};

export const inboxRowPath = (key: string) => `/inbox/${key}`;

// The row the path names, while it is still listed.
export const useSelectedInboxRow = (): InboxRow | undefined => {
  const { data } = useFetchInbox();
  const { rowKey } = useParams();
  return data && rowKey ? inboxRows(data).find((r) => r.key === rowKey) : undefined;
};

// Opening a row puts it in the path and marks every unread notification in it read.
export const useOpenInboxRow = () => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const markRead = useMarkNotificationsRead();
  return (row: InboxRow) => {
    navigate(wsPath(inboxRowPath(row.key)));
    if (row.unreadIds.length > 0) markRead.mutate(row.unreadIds);
  };
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

// Every workspace's unread count at once, for the switcher; shares the badge's key prefix so the same invalidations reach it.
export const useFetchUnreadByWorkspace = () =>
  useQuery({
    queryKey: [getUnreadCountKey],
    queryFn: async () => (await api.get<UnreadCount>("/api/notifications/unread-count")).data,
    select: (d) => d.workspaces,
  });

export const useMarkNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => Promise.all(ids.map((id) => api.post(`/api/notifications/${id}/read`))),
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
