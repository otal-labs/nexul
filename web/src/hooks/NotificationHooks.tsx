import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useInboxStore } from "@/stores/inboxStore";
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

// The row the inbox shows open: the one last chosen while it is still listed, otherwise the newest.
export const useSelectedInboxRow = (): InboxRow | undefined => {
  const { data } = useFetchInbox();
  const key = useInboxStore((s) => s.selectedKey);
  const rows = data ? inboxRows(data) : [];
  return rows.find((r) => r.key === key) ?? rows[0];
};

// Opening a row selects it and marks every unread notification in it read.
export const useOpenInboxRow = () => {
  const select = useInboxStore((s) => s.select);
  const markRead = useMarkNotificationsRead();
  return (row: InboxRow) => {
    select(row.key);
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
