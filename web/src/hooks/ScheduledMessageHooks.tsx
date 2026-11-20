import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type { ScheduleInput, ScheduledMessage } from "@nexul/client-core/chat";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { upsertCachedMessage } from "@/hooks/ChatHooks";
import type { Message } from "@/models/Chat";

export const getScheduledMessagesKey = "getScheduledMessages";

// The viewer's own messages waiting to post in a conversation, soonest first; nobody else sees them.
export const useFetchScheduledMessages = (conversationId: string | undefined) =>
  useQuery({
    queryKey: [getScheduledMessagesKey, conversationId],
    queryFn: async () => (await api.get<ScheduledMessage[]>(`/api/chat/conversations/${conversationId}/scheduled-messages`)).data,
    enabled: !!conversationId,
  });

const useRefetchScheduled = () => {
  const client = useQueryClient();
  return (conversationId: string) => client.invalidateQueries({ queryKey: [getScheduledMessagesKey, conversationId] });
};

export const useScheduleMessage = (conversationId: string) => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("channels:post", async (input: ScheduleInput) =>
      (await api.post<ScheduledMessage>(`/api/chat/conversations/${conversationId}/scheduled-messages`, input)).data),
    onSuccess: () => refetch(conversationId),
  });
};

export const useUpdateScheduledMessage = () => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("none", async ({ id, change }: { id: string; change: Partial<ScheduleInput> }) =>
      (await api.patch<ScheduledMessage>(`/api/chat/scheduled-messages/${id}`, change)).data),
    onSuccess: (saved) => refetch(saved.conversation_id),
  });
};

export const useCancelScheduledMessage = () => {
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("none", async (scheduled: ScheduledMessage) => api.delete(`/api/chat/scheduled-messages/${scheduled.id}`)),
    onSuccess: (_result, scheduled) => refetch(scheduled.conversation_id),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Posts it at once; a repeating one keeps its next send.
export const useSendScheduledMessageNow = () => {
  const client = useQueryClient();
  const refetch = useRefetchScheduled();
  return useMutation({
    mutationFn: requires("channels:post", async (scheduled: ScheduledMessage) =>
      (await api.post<Message>(`/api/chat/scheduled-messages/${scheduled.id}/send`)).data),
    onSuccess: (message) => {
      upsertCachedMessage(client, message);
      return refetch(message.conversation_id);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
