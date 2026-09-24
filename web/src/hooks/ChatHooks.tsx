import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getMeKey } from "@/hooks/AuthHooks";
import { useFetchWorkspaceMembers } from "@/hooks/MemberHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import type { Conversation, Message, UnreadCounts } from "@/models/Chat";
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
export const useFetchChatThreadIndicators = (projectId: string | undefined) => {
  const { data: tickets } = useFetchTicketsByProject(projectId);
  const ids = tickets?.map((t) => t.id) ?? [];
  return useQuery({
    queryKey: [getChatThreadIndicatorsKey, projectId, ids],
    queryFn: async () =>
      (
        await api.get<Record<string, boolean>>("/api/chat/tickets/thread-status", {
          params: { ticket_ids: ids.join(",") },
        })
      ).data,
    enabled: !!projectId && ids.length > 0,
  });
};

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

export const useCreateChannel = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) =>
      (await api.post<Conversation>("/api/chat/channels", { workspace_id: workspaceId, name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] });
      toast.success("Channel created");
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

// A project's interview thread renders inline on its Interview page, loaded once a run has created it.
export const useFetchOrCreateInterviewThread = (workspaceId: string, projectId: string, enabled: boolean) =>
  useQuery({
    queryKey: ["getOrCreateInterviewThread", workspaceId, projectId],
    queryFn: async () =>
      (await api.post<Conversation>(`/api/chat/projects/${projectId}/interview-thread`, { workspace_id: workspaceId }))
        .data,
    enabled: enabled && workspaceId !== "" && projectId !== "",
  });

// The doc thread's Thread button navigates to the chat page (ADR 0060) rather than rendering inline like a ticket's,
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
// A server copy also retires the optimistic row it confirms, whichever of the POST reply or the push lands first.
export const upsertCachedMessage = (client: QueryClient, message: Message) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, message.conversation_id] }, (old) => {
    if (!old) return old;
    const kept = old.filter((m) => !(m.pending && m.author_id === message.author_id && m.body === message.body));
    if (kept.some((m) => m.id === message.id)) return kept.map((m) => (m.id === message.id ? message : m));
    return [...kept, message];
  });

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

// A plain object lookup (one shared members fetch), not a query per message.
export const useChatAuthorLookup = (workspaceId: string) => {
  const { data: membersList } = useFetchWorkspaceMembers(workspaceId);
  const members = membersList?.members ?? [];
  return (authorId: string): string => members.find((m) => m.user_id === authorId)?.login ?? authorId;
};
