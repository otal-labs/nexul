import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchMe } from "@/hooks/AuthHooks";
import { getChatMessagesKey, upsertCachedMessage } from "@/hooks/ChatHooks";
import { applyReaction, type Message } from "@/models/Chat";
import type { LiveFollower } from "@/lib/live";

export interface ReactionChange {
  messageId: string;
  userId: string;
  emoji: string;
  reacted: boolean;
}

export const applyCachedReaction = (client: QueryClient, conversationId: string, change: ReactionChange) =>
  client.setQueriesData<Message[]>({ queryKey: [getChatMessagesKey, conversationId] }, (old) =>
    old?.map((m) =>
      m.id === change.messageId ? { ...m, reactions: applyReaction(m.reactions, change.emoji, change.userId, change.reacted) } : m,
    ),
  );

// Reacting toggles the chip at once; a refused reaction is put back as it was.
export const useReactToMessage = (conversationId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ messageId, emoji, reacted }: ReactionChange) =>
      (await api.put<Message>(`/api/chat/messages/${messageId}/reactions`, { emoji, reacted })).data,
    onMutate: (change) => applyCachedReaction(client, conversationId, change),
    onSuccess: (message) => upsertCachedMessage(client, message),
    onError: (error, change) => {
      applyCachedReaction(client, conversationId, { ...change, reacted: !change.reacted });
      toast.error(errorMessage(error));
    },
  });
};

// useToggleReaction flips the viewer's reaction with an emoji: off when they already reacted with it, on otherwise.
export const useToggleReaction = (message: Message) => {
  const { data: me } = useFetchMe();
  const react = useReactToMessage(message.conversation_id);
  return (emoji: string) => {
    const userId = me?.user.id;
    if (!userId) return;
    const reacted = !(message.reactions ?? []).some((r) => r.emoji === emoji && r.user_ids.includes(userId));
    react.mutate({ messageId: message.id, userId, emoji, reacted });
  };
};

interface ReactionsChangedPayload {
  conversation_id: string;
  message_id: string;
  user_id: string;
  emoji: string;
  reacted: boolean;
}

export const reactionFollower: LiveFollower = {
  "chat.message.reactions_changed": (p: ReactionsChangedPayload, { client }) =>
    applyCachedReaction(client, p.conversation_id, { messageId: p.message_id, userId: p.user_id, emoji: p.emoji, reacted: p.reacted }),
};
