import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import { applyReaction, conversationLabel, upsertMessage, type Conversation, type DMLabelContext, type UnreadCounts } from "@nexul/client-core/chat";

import { api } from "@/api/client";
import { getMeKey } from "@/hooks/AuthHooks";
import { useFetchDoc } from "@/hooks/DocHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { defineQuery } from "@/lib/liveQuery";
import type { Message } from "@/models/Chat";
import { ticketKey } from "@/models/Ticket";
import type { MeResponse } from "@/models/User";

export const getChatConversationsKey = "getChatConversations";
export const getChatMessagesKey = "getChatMessages";
export const getChatUnreadKey = "getChatUnread";

// The server returns the newest page; scrolling further back than this stays on the web for now.
const threadMessageLimit = 100;

const conversationsQuery = defineQuery({
  key: getChatConversationsKey,
  fetch: (workspaceId: string | undefined) =>
    api.get<Conversation[]>(`/api/chat/conversations?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
  refreshes: {
    "chat.conversation.created": { key: (p) => p.conversation.workspace_id },
    "chat.conversation.updated": { key: (p) => p.workspace_id },
    "chat.conversation.deleted": { key: (p) => p.workspace_id },
    // A private channel the viewer lost drops from the list.
    "chat.conversation.members_changed": { key: (p) => p.workspace_id },
  },
});

export const useFetchConversations = (workspaceId: string | undefined) =>
  useQuery({ ...conversationsQuery.options(workspaceId), enabled: !!workspaceId });

const unreadQuery = defineQuery({
  key: getChatUnreadKey,
  fetch: (workspaceId: string | undefined) =>
    api.get<UnreadCounts>(`/api/chat/unread?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
  refreshes: {
    "chat.conversation.deleted": { key: (p) => p.workspace_id },
    "chat.conversation.members_changed": { key: (p) => p.workspace_id },
    "chat.message.created": "all",
    "chat.message.deleted": "all",
  },
});

export const useFetchChatUnread = (workspaceId: string | undefined) =>
  useQuery({ ...unreadQuery.options(workspaceId), enabled: !!workspaceId });

// Message frames carry the change, so an open thread is patched in place and its page never downloads again; a
// deleted channel or a lost private one refetches into its error state instead of showing stale messages.
const messagesQuery = defineQuery({
  key: getChatMessagesKey,
  fetch: (conversationId: string | undefined) =>
    api.get<Message[]>(`/api/chat/conversations/${conversationId}/messages?limit=${threadMessageLimit}`),
  refreshes: {
    "chat.conversation.deleted": { key: (p) => p.conversation_id },
    "chat.conversation.members_changed": { key: (p) => p.conversation_id },
    // The catalog types a message as an open object; it is the same Message the thread route returns.
    "chat.message.created": { patch: (client, p) => upsertCachedMessage(client, p.message as unknown as Message) },
    "chat.message.updated": { patch: (client, p) => upsertCachedMessage(client, p.message as unknown as Message) },
    "chat.message.deleted": { patch: (client, p) => markCachedMessageDeleted(client, p.conversation_id, p.message_id, p.deleted_at) },
    "chat.message.reactions_changed": { patch: (client, p) => applyCachedReaction(client, p) },
  },
});

export const useFetchMessages = (conversationId: string | undefined) =>
  useQuery({ ...messagesQuery.options(conversationId), enabled: !!conversationId });

// A ticket thread reads "KEY-N Title" and a doc thread reads the doc's title, like the web; every other kind keeps its own label.
export const useConversationLabel = (conversation: Conversation, dmCtx: DMLabelContext): string => {
  const { data: ticket } = useFetchTicket(conversation.kind === "ticket_thread" ? conversation.ticket_id : undefined);
  const { data: project } = useFetchProject(ticket?.project_id);
  const { data: doc } = useFetchDoc(conversation.kind === "doc_thread" ? conversation.doc_id : undefined);
  if (ticket) return `${ticketKey(ticket, project?.prefix)} ${ticket.title}`;
  if (doc) return doc.title;
  return conversationLabel(conversation, dmCtx);
};

export const upsertCachedMessage = (client: QueryClient, message: Message) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, message.conversation_id] }, (old) => old && upsertMessage(old, message));

const removeCachedMessage = (client: QueryClient, conversationId: string, messageId: string) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.filter((m) => m.id !== messageId),
  );

const patchCachedMessage = (client: QueryClient, conversationId: string, messageId: string, patch: (m: Message) => Message) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.map((m) => (m.id === messageId ? patch(m) : m)),
  );

export const markCachedMessageDeleted = (client: QueryClient, conversationId: string, messageId: string, deletedAt: string) =>
  patchCachedMessage(client, conversationId, messageId, (m) => ({ ...m, deleted_at: deletedAt }));

export interface ReactionChange {
  conversation_id: string;
  message_id: string;
  user_id: string;
  emoji: string;
  reacted: boolean;
}

export const applyCachedReaction = (client: QueryClient, change: ReactionChange) =>
  patchCachedMessage(client, change.conversation_id, change.message_id, (m) => ({
    ...m,
    reactions: applyReaction(m.reactions, change.emoji, change.user_id, change.reacted),
  }));

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
  const workspaceId = useCurrentWorkspaceId();
  useEffect(() => {
    if (!conversationId || !lastMessageId) return;
    void api
      .post<null>(`/api/chat/conversations/${conversationId}/read`)
      .then(() => client.invalidateQueries({ queryKey: [getChatUnreadKey, workspaceId] }))
      .catch(() => undefined);
  }, [client, conversationId, lastMessageId, workspaceId]);
};
