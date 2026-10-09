import { useMutation, useQuery, useQueryClient, type QueryClient, type QueryKey } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { DocFolder } from "@/models/DocFolder";
import { SubjectType, type Notification, type UnreadCount } from "@/models/Notification";
import { groupInbox, inboxRows, type InboxRow } from "@/utils/InboxUtility";
import { refetchHolding, type LiveFollower } from "@/lib/live";

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

const refetchInboxes = (client: QueryClient, holds: (row: Notification) => boolean) =>
  refetchHolding(client, [getNotificationsKey], (rows: Notification[]) => rows.some(holds));

interface FolderPayload {
  folder: DocFolder;
}

// notification.created reaches only its recipients and names the workspace whose inbox holds the new notices.
interface NotificationCreatedPayload {
  workspace_id: string;
}

// The inbox and badge of that workspace, and the views across every workspace, which hold it too.
const showsWorkspace = (workspaceId: string) => ({ queryKey }: { queryKey: QueryKey }) => !queryKey[1] || queryKey[1] === workspaceId;

// The inbox groups doc rows by the folder each doc is in now, so a move, rename, or delete regroups them.
export const notificationFollower: LiveFollower = {
  "notification.created": ({ workspace_id }: NotificationCreatedPayload, { client }) =>
    Promise.all([getNotificationsKey, getUnreadCountKey].map((key) => client.invalidateQueries({ queryKey: [key], predicate: showsWorkspace(workspace_id) }))),
  "doc.moved": ({ doc }: { doc: { id: string } }, { client }) => refetchInboxes(client, (n) => n.subject_type === SubjectType.Doc && n.subject_id === doc.id),
  "doc.folder.updated": ({ folder }: FolderPayload, { client }) =>
    client.setQueriesData<Notification[]>({ queryKey: [getNotificationsKey] }, (rows) =>
      rows?.some((n) => n.folder_id === folder.id) ? rows.map((n) => (n.folder_id === folder.id ? { ...n, folder_name: folder.name, folder_is_default: folder.is_default } : n)) : rows,
    ),
  "doc.folder.deleted": ({ folder }: FolderPayload, { client }) => refetchInboxes(client, (n) => n.folder_id === folder.id),
};
