import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { toast } from "sonner";

import type { Conversation } from "@nexul/client-core/chat";

import { api, errorMessage } from "@/api/client";
import { conversationPlayTarget } from "@/models/Chat";
import type { PlayType } from "@/models/Play";
import type { PlayQueue, PlayQueueItem } from "@/models/PlayQueue";
import { hasPlayQueue, threadQueueEvents, type ThreadQueueEvent } from "@/utils/PlayQueueUtility";
import { followEach, type LiveFollower } from "@/lib/live";

export const getPlayQueueKey = "getPlayQueue";
export const getPlayQueuedKey = "getPlayQueued";

// The rolling day freeing a paused target sends no frame, so the query looks again once its oldest counted run leaves.
const lookAgainWhenFreed = (queue: PlayQueue | undefined): number | false =>
  queue?.paused_until ? Math.max(Date.parse(queue.paused_until) - Date.now(), 0) + 1000 : false;

// A ticket's or doc's auto runs and whether they are paused; the queue frames keep it live.
export const useFetchPlayQueue = (targetType: PlayType, targetId: string) =>
  useQuery({
    queryKey: [getPlayQueueKey, targetType, targetId],
    queryFn: async () =>
      (await api.get<PlayQueue>("/api/plays/queue", { params: { target_type: targetType, target_id: targetId } })).data,
    enabled: targetId !== "" && hasPlayQueue(targetType),
    refetchInterval: (query) => lookAgainWhenFreed(query.state.data),
  });

export const useFetchPlayQueued = (playId: string) =>
  useQuery({
    queryKey: [getPlayQueuedKey, playId],
    queryFn: async () => (await api.get<PlayQueueItem[]>("/api/plays/queued", { params: { play_id: playId } })).data,
    enabled: playId !== "",
  });

export const useThreadQueueEvents = (conversation: Conversation): ThreadQueueEvent[] => {
  const target = conversationPlayTarget(conversation);
  const { data } = useFetchPlayQueue(target?.type ?? "ticket", target?.id ?? "");
  return useMemo(() => threadQueueEvents(data), [data]);
};

const refreshQueues = (client: ReturnType<typeof useQueryClient>, item: PlayQueueItem) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getPlayQueueKey, item.target_type, item.target_id], exact: true }),
    client.invalidateQueries({ queryKey: [getPlayQueuedKey, item.play_id], exact: true }),
  ]);

export const useCancelQueuedRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (itemId: string) => (await api.post<PlayQueueItem>(`/api/plays/queue/${itemId}/cancel`)).data,
    onSuccess: async (item) => {
      await refreshQueues(client, item);
      toast.success(`${item.play_label} cancelled`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useResumeAutoPlays = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ targetType, targetId }: { targetType: PlayType; targetId: string }) =>
      (await api.post<PlayQueue>("/api/plays/queue/resume", { target_type: targetType, target_id: targetId })).data,
    onSuccess: (queue, { targetType, targetId }) => {
      client.setQueryData([getPlayQueueKey, targetType, targetId], queue);
      toast.success("Auto plays resumed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// play.queued and play.queue_updated carry the item; play.queue_resumed names only its target.
type QueueFrame = Pick<PlayQueueItem, "target_type" | "target_id">;

export const playQueueFollower: LiveFollower = {
  ...followEach(["play.queued", "play.queue_updated"], (item: PlayQueueItem, { client }) => refreshQueues(client, item)),
  "play.queue_resumed": ({ target_type, target_id }: QueueFrame, { client }) =>
    client.invalidateQueries({ queryKey: [getPlayQueueKey, target_type, target_id], exact: true }),
};
