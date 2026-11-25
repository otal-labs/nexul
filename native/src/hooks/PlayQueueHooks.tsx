import { useMutation, useQueries, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import { isQueued, type PlayQueue, type PlayQueueItem, type PlayType } from "@/models/Play";

export const getPlayQueueKey = "getPlayQueue";
export const getPlayQueuedKey = "getPlayQueued";

// A queue frame names its target, so it refetches that target's queue alone: its paused flag and counts aren't in the frame.
const refetchTargetQueue = (client: QueryClient, frame: Pick<PlayQueueItem, "target_type" | "target_id">) =>
  void client.invalidateQueries({ queryKey: [getPlayQueueKey, frame.target_type, frame.target_id], exact: true });

// A ticket's or doc's auto runs, and whether the daily cap paused them.
const playQueueQuery = defineQuery({
  key: getPlayQueueKey,
  fetch: (targetType: PlayType, targetId: string) => api.get<PlayQueue>(`/api/plays/queue?target_type=${targetType}&target_id=${targetId}`),
  refreshes: {
    "play.queued": { patch: refetchTargetQueue },
    "play.queue_updated": { patch: refetchTargetQueue },
    "play.queue_resumed": { patch: refetchTargetQueue },
  },
});

// An interview has no queue; a run there starts only when pressed.
export const useFetchPlayQueue = (targetType: PlayType, targetId: string) =>
  useQuery({ ...playQueueQuery.options(targetType, targetId), enabled: targetId !== "" && targetType !== "interview" });

// The frame carries the item, so it is written into its play's list: kept while it waits, dropped once decided.
const upsertQueued = (client: QueryClient, item: PlayQueueItem) =>
  void client.setQueriesData<PlayQueueItem[]>({ queryKey: [getPlayQueuedKey, item.play_id], exact: true }, (items) => {
    if (!items) return items;
    const rest = items.filter((queued) => queued.id !== item.id);
    return isQueued(item) ? [...rest, item] : rest;
  });

// What waits in the queue for one play.
const playQueuedQuery = defineQuery({
  key: getPlayQueuedKey,
  fetch: (playId: string) => api.get<PlayQueueItem[]>(`/api/plays/queued?play_id=${playId}`),
  refreshes: {
    "play.queued": { patch: upsertQueued },
    "play.queue_updated": { patch: upsertQueued },
    // A resume names only its target, which any play's waiting runs may be on.
    "play.queue_resumed": "all",
  },
});

export const useFetchPlayQueued = (playId: string) => useQuery({ ...playQueuedQuery.options(playId), enabled: playId !== "" });

// What waits in the queue across these plays, oldest first.
export const useQueuedAcross = (playIds: string[]) =>
  useQueries({
    queries: playIds.map((playId) => playQueuedQuery.options(playId)),
    combine: (results) => results.flatMap((result) => result.data ?? []).sort((a, b) => a.queued_at.localeCompare(b.queued_at)),
  });

export const useCancelQueuedRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("autoplays:write", (itemId: string) => api.post<PlayQueueItem>(`/api/plays/queue/${itemId}/cancel`)),
    onSuccess: (item) => {
      upsertQueued(client, item);
      refetchTargetQueue(client, item);
    },
  });
};

export const useResumeAutoPlays = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("autoplays:write", ({ targetType, targetId }: { targetType: PlayType; targetId: string }) =>
      api.post<PlayQueue>("/api/plays/queue/resume", { target_type: targetType, target_id: targetId }),
    ),
    onSuccess: (queue, { targetType, targetId }) => client.setQueryData([getPlayQueueKey, targetType, targetId], queue),
  });
};
