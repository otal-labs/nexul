import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { toast } from "sonner";

import type { Conversation } from "@nexul/client-core/chat";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getActiveTrailsKey } from "@/hooks/TrailHooks";
import { conversationPlayTarget } from "@/models/Chat";
import type { PlayType } from "@/models/Play";
import type { PlayQueue, PlayQueueItem } from "@/models/PlayQueue";
import { hasPlayQueue, threadQueueEvents, type ThreadQueueEvent } from "@/utils/PlayQueueUtility";
import { followEach, type LiveFollower } from "@/lib/live";

export const getPlayQueueKey = "getPlayQueue";
export const getPlayQueuedKey = "getPlayQueued";

// A ticket's or doc's auto runs; the queue frames keep it live.
export const useFetchPlayQueue = (targetType: PlayType, targetId: string) =>
  useQuery({
    queryKey: [getPlayQueueKey, targetType, targetId],
    queryFn: async () =>
      (await api.get<PlayQueue>("/api/plays/queue", { params: { target_type: targetType, target_id: targetId } })).data,
    enabled: targetId !== "" && hasPlayQueue(targetType),
  });

export const useFetchPlayQueued = (playId: string) =>
  useQuery({
    queryKey: [getPlayQueuedKey, playId],
    queryFn: async () => (await api.get<PlayQueueItem[]>("/api/plays/queued", { params: { play_id: playId } })).data,
    enabled: playId !== "",
  });

export const useQueueEvents = (targetType: PlayType, targetId: string): ThreadQueueEvent[] => {
  const { data } = useFetchPlayQueue(targetType, targetId);
  return useMemo(() => threadQueueEvents(data), [data]);
};

export const useThreadQueueEvents = (conversation: Conversation): ThreadQueueEvent[] => {
  const target = conversationPlayTarget(conversation);
  return useQueueEvents(target?.type ?? "ticket", target?.id ?? "");
};

const refreshQueues = (client: ReturnType<typeof useQueryClient>, item: PlayQueueItem) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getPlayQueueKey, item.target_type, item.target_id], exact: true }),
    client.invalidateQueries({ queryKey: [getPlayQueuedKey, item.play_id], exact: true }),
  ]);

export const useCancelQueuedRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("autoplays:write", async (itemId: string) => (await api.post<PlayQueueItem>(`/api/plays/queue/${itemId}/cancel`)).data),
    onSuccess: async (item) => {
      await refreshQueues(client, item);
      toast.success(`${item.play_label} cancelled`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// play.queued and play.queue_updated carry the item.
// The board's batch answer names the queued targets, so a queue frame refreshes it too.
export const playQueueFollower: LiveFollower = {
  ...followEach(["play.queued", "play.queue_updated"], (item: PlayQueueItem, { client }) =>
    Promise.all([
      refreshQueues(client, item),
      client.invalidateQueries({ queryKey: [getActiveTrailsKey, item.target_type, item.project_id], exact: true }),
    ])),
};
