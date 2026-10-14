import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { toast } from "sonner";

import type { Conversation } from "@nexul/client-core/chat";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { invalidateProjectLinks } from "@/hooks/PairingProjectHooks";
import { conversationPlayTarget } from "@/models/Chat";
import type { PlayType } from "@/models/Play";
import type { PlayQueue, PlayQueueItem } from "@/models/PlayQueue";
import { hasPlayQueue, threadQueueEvents, withoutErrorPrefix, type ThreadQueueEvent } from "@/utils/PlayQueueUtility";
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

interface PickComputerInput {
  itemId: string;
  computerId: string;
  harnessProjectId: string;
}

// Runs a queued run on one of its person's computers and a T3 project there, saved as their link for its project.
export const usePickQueuedComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async ({ itemId, computerId, harnessProjectId }: PickComputerInput) =>
      (
        await api.post<PlayQueueItem>(`/api/plays/queue/${itemId}/computer`, {
          computer_id: computerId,
          harness_project_id: harnessProjectId,
        })
      ).data),
    onSuccess: async (item) => {
      await Promise.all([refreshQueues(client, item), invalidateProjectLinks(client)]);
      toast.success(`${item.play_label} starts on the picked computer, now linked for this project`);
    },
    onError: (error) => toast.error(withoutErrorPrefix(errorMessage(error))),
  });
};

export const useResumeAutoPlays = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("autoplays:write", async ({ targetType, targetId }: { targetType: PlayType; targetId: string }) =>
      (await api.post<PlayQueue>("/api/plays/queue/resume", { target_type: targetType, target_id: targetId })).data),
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
