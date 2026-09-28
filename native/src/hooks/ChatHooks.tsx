import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { api } from "@/api/client";
import { getMeKey } from "@/hooks/AuthHooks";
import type { Conversation, Message, UnreadCounts, Workspace, WorkspaceMembers } from "@/models/Chat";
import type { MeResponse } from "@/models/User";

export const getChatWorkspacesKey = "getWorkspaces";
export const getChatMembersKey = "getWorkspaceMembers";
export const getChatConversationsKey = "getChatConversations";
export const getChatMessagesKey = "getChatMessages";
export const getChatUnreadKey = "getChatUnread";

// ponytail: the list endpoint returns the oldest N, so a thread past this many messages loses its newest; needs a newest-first page on the server.
const threadMessageLimit = 500;

// ponytail: first workspace until the Your settings switcher lands and scopes every tab.
export const useChatWorkspaceId = (): string | undefined => {
  const { data: workspaces } = useQuery({
    queryKey: [getChatWorkspacesKey],
    queryFn: () => api.get<Workspace[]>("/api/workspaces"),
  });
  return workspaces?.[0]?.id;
};

export const useFetchConversations = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getChatConversationsKey, workspaceId],
    queryFn: () =>
      api.get<Conversation[]>(`/api/chat/conversations?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
    enabled: !!workspaceId,
  });

export const useFetchChatUnread = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getChatUnreadKey, workspaceId],
    queryFn: () => api.get<UnreadCounts>(`/api/chat/unread?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
    enabled: !!workspaceId,
  });

export const useFetchMessages = (conversationId: string | undefined) =>
  useQuery({
    queryKey: [getChatMessagesKey, conversationId],
    queryFn: () =>
      api.get<Message[]>(`/api/chat/conversations/${conversationId}/messages?limit=${threadMessageLimit}`),
    enabled: !!conversationId,
  });

// Author names for message rows and DM labels: one members fetch, looked up per message.
export const useChatAuthorLookup = (workspaceId: string | undefined) => {
  const { data } = useQuery({
    queryKey: [getChatMembersKey, workspaceId],
    queryFn: () => api.get<WorkspaceMembers>(`/api/workspaces/${workspaceId}/members`),
    enabled: !!workspaceId,
  });
  const members = data?.members ?? [];
  return (userId: string): string => members.find((m) => m.user_id === userId)?.login ?? userId;
};

// The server copy retires the optimistic row it confirms, whichever of the POST reply or the refetch lands first.
export const upsertCachedMessage = (client: QueryClient, message: Message) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, message.conversation_id] }, (old) => {
    if (!old) return old;
    const kept = old.filter((m) => !(m.pending && m.author_id === message.author_id && m.body === message.body));
    if (kept.some((m) => m.id === message.id)) return kept.map((m) => (m.id === message.id ? message : m));
    return [...kept, message];
  });

const removeCachedMessage = (client: QueryClient, conversationId: string, messageId: string) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.filter((m) => m.id !== messageId),
  );

export const usePostMessage = (conversationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api.post<Message>(`/api/chat/conversations/${conversationId}/messages`, { body }),
    onMutate: async (body) => {
      await client.cancelQueries({ queryKey: [getChatMessagesKey, conversationId] });
      const now = new Date().toISOString();
      const pending: Message = {
        id: `pending-${now}`,
        conversation_id: conversationId,
        author_id: client.getQueryData<MeResponse>([getMeKey])?.user.id ?? "",
        author_kind: "user",
        body,
        mentions: null,
        created_at: now,
        updated_at: now,
        pending: true,
      };
      upsertCachedMessage(client, pending);
      return { pendingId: pending.id };
    },
    onSuccess: (message) => upsertCachedMessage(client, message),
    onError: (_error, _body, context) => {
      if (context) removeCachedMessage(client, conversationId, context.pendingId);
    },
  });
};

// Marks the thread read on open and on each new message; a plain call, since a mutation observer would only re-render the thread.
export const useMarkThreadRead = (conversationId: string, lastMessageId: string | undefined) => {
  const client = useQueryClient();
  const workspaceId = useChatWorkspaceId();
  useEffect(() => {
    if (!conversationId || !lastMessageId) return;
    void api
      .post<null>(`/api/chat/conversations/${conversationId}/read`)
      .then(() => client.invalidateQueries({ queryKey: [getChatUnreadKey, workspaceId] }))
      .catch(() => undefined);
  }, [client, conversationId, lastMessageId, workspaceId]);
};
