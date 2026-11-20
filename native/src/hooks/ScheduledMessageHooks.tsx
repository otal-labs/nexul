import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ScheduleInput, ScheduledMessage } from "@nexul/client-core/chat";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { upsertCachedMessage } from "@/hooks/ChatHooks";
import { defineQuery } from "@/lib/liveQuery";
import type { Message } from "@/models/Chat";

export const getScheduledMessagesKey = "getScheduledMessages";

// The viewer's own messages waiting to post in a conversation, soonest first; the frame reaches only their author.
const scheduledQuery = defineQuery({
  key: getScheduledMessagesKey,
  fetch: (conversationId: string | undefined) => api.get<ScheduledMessage[]>(`/api/chat/conversations/${conversationId}/scheduled-messages`),
  refreshes: {
    "chat.scheduled_message.changed": { key: (p) => p.conversation_id },
  },
});

export const useFetchScheduledMessages = (conversationId: string | undefined) =>
  useQuery({ ...scheduledQuery.options(conversationId), enabled: !!conversationId });

const useRefetchScheduled = () => {
  const client = useQueryClient();
  return (conversationId: string) => client.invalidateQueries({ queryKey: [getScheduledMessagesKey, conversationId] });
};

export const useScheduleMessage = (conversationId: string) => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("channels:post", (input: ScheduleInput) =>
      api.post<ScheduledMessage>(`/api/chat/conversations/${conversationId}/scheduled-messages`, input)),
    onSuccess: () => refetch(conversationId),
  });
};

export const useUpdateScheduledMessage = () => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("none", ({ id, change }: { id: string; change: Partial<ScheduleInput> }) =>
      api.patch<ScheduledMessage>(`/api/chat/scheduled-messages/${id}`, change)),
    onSuccess: (saved) => refetch(saved.conversation_id),
  });
};

export const useCancelScheduledMessage = () => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("none", (scheduled: ScheduledMessage) => api.delete<null>(`/api/chat/scheduled-messages/${scheduled.id}`)),
    onSuccess: (_result, scheduled) => refetch(scheduled.conversation_id),
  });
};

// Posts it at once; a repeating one keeps its next send.
export const useSendScheduledMessageNow = () => {
  const client = useQueryClient();
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("channels:post", (scheduled: ScheduledMessage) => api.post<Message>(`/api/chat/scheduled-messages/${scheduled.id}/send`)),
    onSuccess: (message) => {
      upsertCachedMessage(client, message);
      return refetch(message.conversation_id);
    },
  });
};
