import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate, type Location, type NavigateFunction } from "react-router";
import { toast } from "sonner";

import { upsertMessage, type Conversation, type UnreadCounts } from "@nexul/client-core/chat";

import { api, errorMessage } from "@/api/client";
import { getMeKey } from "@/hooks/AuthHooks";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { channelMention, type ConversationDeleted, type CreateChannelFormData, type Message } from "@/models/Chat";
import type { QuestionAnswers } from "@/models/Question";
import type { MeResponse } from "@/models/User";

export const getChatConversationsKey = "getChatConversations";
export const getChatMessagesKey = "getChatMessages";
export const getChatUnreadKey = "getChatUnread";
export const getChatThreadIndicatorsKey = "getChatThreadIndicators";
export const getChatTicketThreadStatusKey = "getChatTicketThreadStatus";

export const useFetchConversations = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getChatConversationsKey, workspaceId],
    queryFn: async () =>
      (await api.get<Conversation[]>("/api/chat/conversations", { params: { workspace_id: workspaceId } })).data,
    enabled: !!workspaceId,
  });

export const useFetchMessages = (conversationId: string | undefined, limit?: number) =>
  useQuery({
    queryKey: [getChatMessagesKey, conversationId, limit],
    queryFn: async () =>
      (
        await api.get<Message[]>(`/api/chat/conversations/${conversationId}/messages`, {
          params: limit ? { limit } : undefined,
        })
      ).data,
    enabled: !!conversationId,
  });

export const useFetchChatUnread = (workspaceId: string | undefined, enabled = true) =>
  useQuery({
    queryKey: [getChatUnreadKey, workspaceId],
    queryFn: async () =>
      (await api.get<UnreadCounts>("/api/chat/unread", { params: { workspace_id: workspaceId } })).data,
    enabled: enabled && !!workspaceId,
  });

// Batched per project: every TicketCard shares the queryKey, so a 40-card board fires one request, not 40.
export const useFetchChatThreadIndicators = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getChatThreadIndicatorsKey, projectId],
    queryFn: async () =>
      (await api.get<Record<string, boolean>>("/api/chat/tickets/thread-status", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

// Read-only "does a thread exist" check, so an existing thread shows without ever calling get-or-create.
export const useFetchTicketThreadStatus = (ticketId: string | undefined) =>
  useQuery({
    queryKey: [getChatTicketThreadStatusKey, ticketId],
    queryFn: async () =>
      (
        await api.get<Record<string, boolean>>("/api/chat/tickets/thread-status", {
          params: { ticket_ids: ticketId },
        })
      ).data,
    enabled: !!ticketId,
  });

// A public channel sends only its name; a private one adds who starts in it besides the creator.
export const useCreateChannel = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ voice, name, private: isPrivate, member_ids }: CreateChannelFormData & { voice: boolean }) =>
      (
        await api.post<Conversation>(voice ? "/api/chat/voice-channels" : "/api/chat/channels", {
          workspace_id: workspaceId,
          name,
          ...(isPrivate && { private: true, member_ids }),
        })
      ).data,
    onSuccess: async (created) => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] });
      toast.success(`${channelMention(created)} created`);
    },
  });
};

export const useRenameChannel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, name }: { id: string; name: string }) =>
      (await api.patch<Conversation>(`/api/chat/conversations/${id}`, { name })).data,
    onSuccess: async (renamed) => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, renamed.workspace_id] });
      toast.success(`Renamed to ${channelMention(renamed)}`);
    },
  });
};

// One toast per deleted conversation, whether the deleter's own reply or the live frame lands first.
const deletedToastId = (conversationId: string) => `conversation-deleted-${conversationId}`;

// Sends a viewer of the conversation to the chat home; false when they are looking at something else.
export const leaveConversationPage = (navigate: NavigateFunction, location: Pick<Location, "pathname">, conversationId: string) => {
  if (!location.pathname.endsWith(`/chat/${conversationId}`)) return false;
  void navigate(location.pathname.slice(0, -conversationId.length - 1), { replace: true });
  return true;
};

// Leaves a conversation that was just deleted: its call ends and a viewer of it lands on the chat home.
export const followConversationDeleted = (
  navigate: NavigateFunction,
  location: Pick<Location, "pathname">,
  deleted: Pick<ConversationDeleted, "conversation_id" | "kind" | "name">,
) => {
  const call = useVoiceCallStore.getState();
  if (call.activeConversationId === deleted.conversation_id) call.leave();
  if (!leaveConversationPage(navigate, location, deleted.conversation_id)) return;
  toast.info(`${channelMention(deleted)} was deleted`, { id: deletedToastId(deleted.conversation_id) });
};

export const useDeleteChannel = () => {
  const client = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  return useMutation({
    mutationFn: async (conversation: Conversation) => api.delete(`/api/chat/conversations/${conversation.id}`),
    onSuccess: async (_result, conversation) => {
      toast.success(`${channelMention(conversation)} deleted`, { id: deletedToastId(conversation.id) });
      followConversationDeleted(navigate, location, { conversation_id: conversation.id, kind: conversation.kind, name: conversation.name ?? "" });
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, conversation.workspace_id] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCreateDM = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (participantIds: string[]) =>
      (await api.post<Conversation>("/api/chat/dms", { workspace_id: workspaceId, participant_ids: participantIds }))
        .data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] });
      toast.success("Direct message started");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Wraps the idempotent get-or-create endpoint as a query, not a mutation, so no create-on-mount effect (F5).
export const useFetchOrCreateTicketThread = (workspaceId: string, ticketId: string | undefined, enabled: boolean) =>
  useQuery({
    queryKey: ["getOrCreateTicketThread", workspaceId, ticketId],
    queryFn: async () =>
      (await api.post<Conversation>(`/api/chat/tickets/${ticketId}/thread`, { workspace_id: workspaceId })).data,
    enabled: enabled && !!ticketId,
  });

// The doc thread's Thread button navigates to the chat page (ADR 0093) rather than rendering inline like a ticket's,
// so this is a mutation triggered by the click, not a query loaded on mount.
export const useGetOrCreateDocThread = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (docId: string) =>
      (await api.post<Conversation>(`/api/chat/docs/${docId}/thread`, { workspace_id: workspaceId })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Mutation results and push frames land straight in the cache; refetching the list after every message lagged and flickered.
export const upsertCachedMessage = (client: QueryClient, message: Message) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, message.conversation_id] }, (old) => old && upsertMessage(old, message));

export const removeCachedMessage = (client: QueryClient, conversationId: string, messageId: string) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.filter((m) => m.id !== messageId),
  );

export const markCachedMessageDeleted = (client: QueryClient, conversationId: string, messageId: string, deletedAt: string) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.map((m) => (m.id === messageId ? { ...m, deleted_at: deletedAt } : m)),
  );

export const usePostMessage = (conversationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (body: string) =>
      (await api.post<Message>(`/api/chat/conversations/${conversationId}/messages`, { body })).data,
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
    onError: (error, _body, context) => {
      if (context) removeCachedMessage(client, conversationId, context.pendingId);
      toast.error(errorMessage(error));
    },
  });
};

export const useEditMessage = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ messageId, body }: { messageId: string; body: string }) =>
      (await api.patch<Message>(`/api/chat/messages/${messageId}`, { body })).data,
    onSuccess: (message) => upsertCachedMessage(client, message),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteMessage = (conversationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (messageId: string) => api.delete(`/api/chat/messages/${messageId}`),
    onSuccess: (_result, messageId) => markCachedMessageDeleted(client, conversationId, messageId, new Date().toISOString()),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMarkChatRead = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (conversationId: string) => api.post(`/api/chat/conversations/${conversationId}/read`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getChatUnreadKey, workspaceId] });
    },
  });
};

// The stream bubble clears when the real chat.message.created frame arrives, not on this call's success.
export const useInterruptAgentTurn = (conversationId: string) =>
  useMutation({
    mutationFn: async () => api.post(`/api/agent/conversations/${conversationId}/interrupt`),
    onError: (error) => toast.error(errorMessage(error)),
  });

// The answer is posted as the caller's message by the server and hands the turn its reply; the thread updates live.
export const useAnswerAgentQuestion = (conversationId: string) =>
  useMutation({
    mutationFn: async ({ requestId, answers }: { requestId: string; answers: QuestionAnswers }) =>
      api.post(`/api/agent/conversations/${conversationId}/answer`, { request_id: requestId, answers }),
    onError: (error) => toast.error(errorMessage(error)),
  });
