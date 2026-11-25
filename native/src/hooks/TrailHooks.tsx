import { useMutation, useQueries, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import type { AnswerValue } from "@nexul/client-core/question";
import { requires } from "@nexul/client-core/requires";
import { isTrailActive, mergeLiveStep, type TrailState } from "@nexul/client-core/trail";

import { api } from "@/api/client";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useActionState } from "@/hooks/useActionState";
import { defineQuery } from "@/lib/liveQuery";
import type { PlayType, WorkspacePlay } from "@/models/Play";
import { applyRunFrame, type PlayRun, type PlayRunsPage, type RunFrame, type Trail } from "@/models/Trail";

export const getTrailsKey = "getTrails";
export const getTrailKey = "getTrail";
export const getPlayRunsKey = "getPlayRuns";

// The last state each run's frame carried, so one frame answers "did the state move" the same for every query it reaches.
const seenStates = new Map<string, TrailState>();
const verdicts = new WeakMap<RunFrame, boolean>();

// A run's first frame counts as a move, since nothing says what its state was before.
export const runStateChanged = (frame: RunFrame): boolean => {
  const known = verdicts.get(frame);
  if (known !== undefined) return known;
  const changed = seenStates.get(frame.trail_id) !== frame.state;
  seenStates.set(frame.trail_id, frame.state);
  verdicts.set(frame, changed);
  return changed;
};

// A list row keeps only its newest step, which is all it shows.
const patchRow = <T extends Trail>(row: T, frame: RunFrame): T =>
  applyRunFrame(row, frame, frame.activity ? [frame.activity] : row.activity);

// A frame for a run the list holds patches that row alone; a run it doesn't hold yet refetches the list for its order.
const patchTargetRuns = (client: QueryClient, frame: RunFrame) => {
  const queryKey = [getTrailsKey, frame.target_type, frame.target_id];
  const runs = client.getQueryData<Trail[]>(queryKey);
  if (!runs) return;
  if (!runs.some((run) => run.id === frame.trail_id)) return void client.invalidateQueries({ queryKey, exact: true });
  client.setQueryData<Trail[]>(queryKey, runs.map((run) => (run.id === frame.trail_id ? patchRow(run, frame) : run)));
};

// A ticket's or doc's runs, newest first, each with its newest step.
const targetRunsQuery = defineQuery({
  key: getTrailsKey,
  fetch: (targetType: PlayType, targetId: string) => api.get<Trail[]>(`/api/plays/runs?target_type=${targetType}&target_id=${targetId}&steps=last`),
  refreshes: { "play.run": { patch: patchTargetRuns } },
});

export const useFetchTargetRuns = (targetType: PlayType, targetId: string) =>
  useQuery({ ...targetRunsQuery.options(targetType, targetId), enabled: targetId !== "" });

// The run's screen follows its frames alone: each carries the newest step, the state, the question and how it ended.
const trailQuery = defineQuery({
  key: getTrailKey,
  fetch: (trailId: string) => api.get<Trail>(`/api/plays/runs/${trailId}`),
  refreshes: {
    "play.run": {
      patch: (client, frame) =>
        void client.setQueryData<Trail>([getTrailKey, frame.trail_id], (trail) => trail && applyRunFrame(trail, frame, mergeLiveStep(trail.activity, frame.activity))),
    },
  },
});

export const useFetchTrail = (trailId: string | undefined) => useQuery({ ...trailQuery.options(trailId ?? ""), enabled: !!trailId });

// A page keeps its identity unless it holds the run, so a frame re-renders that run's row alone.
const patchPage = (page: PlayRunsPage | undefined, frame: RunFrame): PlayRunsPage | undefined => {
  if (!page?.items.some((run) => run.id === frame.trail_id)) return page;
  return { ...page, items: page.items.map((run) => (run.id === frame.trail_id ? patchRow(run, frame) : run)) };
};

// A state move can take a run into or out of a page (a new run, an answer given), so it refetches that play's pages; a step patches its row.
const patchPlayRuns = (client: QueryClient, frame: RunFrame) => {
  const queryKey = [getPlayRunsKey, frame.play_id];
  if (runStateChanged(frame)) return void client.invalidateQueries({ queryKey });
  client.setQueriesData<PlayRunsPage>({ queryKey }, (page) => patchPage(page, frame));
};

// One play's runs on every target, newest first, or only those waiting on an answer, oldest first.
const playRunsQuery = defineQuery({
  key: getPlayRunsKey,
  fetch: (playId: string, waiting: boolean, limit: number) =>
    api.get<PlayRunsPage>(`/api/plays/runs?play_id=${playId}&limit=${limit}${waiting ? "&state=waiting" : ""}`),
  refreshes: { "play.run": { patch: patchPlayRuns } },
});

export const useFetchPlayRuns = (playId: string, waiting: boolean, limit: number) =>
  useQuery({ ...playRunsQuery.options(playId, waiting, limit), enabled: playId !== "" });

// shortcut: a running run older than the newest 20 of its play is left out; a workspace-wide active-runs route would lift it.
const NOW_PAGE = 20;

const byNewest = (a: PlayRun, b: PlayRun) => b.started_at.localeCompare(a.started_at);

// What runs across the workspace now: each busy play's newest runs and those waiting on an answer, only the ones still going.
export const useNowRuns = (plays: WorkspacePlay[]) =>
  useQueries({
    queries: plays.flatMap((play) => [
      ...(play.running > 0 ? [playRunsQuery.options(play.id, false, NOW_PAGE)] : []),
      ...(play.waiting > 0 ? [playRunsQuery.options(play.id, true, NOW_PAGE)] : []),
    ]),
    combine: (results) => {
      const runs = new Map<string, PlayRun>();
      results.forEach((result) => result.data?.items.forEach((run) => isTrailActive(run.state) && runs.set(run.id, run)));
      return { runs: [...runs.values()].sort(byNewest), isPending: results.some((result) => result.isPending) };
    },
  });

// The run's starter, or a plays:write holder, may answer, stop or continue it; anyone else reads it.
export const useCanActOnRun = (trail: Pick<Trail, "starter_id" | "project_id">): boolean => {
  const { data: me } = useFetchMe(true);
  const canWrite = useActionState({ permission: "plays:write", projectId: trail.project_id }).state === "live";
  return trail.starter_id === me?.user.id || canWrite;
};

// What the run's own record and its target's list hold after a write that answers with the run.
const settleRun = (client: QueryClient, trail: Trail) => {
  client.setQueryData([getTrailKey, trail.id], trail);
  return client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id], exact: true });
};

export interface RunPlayInput {
  target_type: PlayType;
  target_id: string;
  custom_instructions: string;
}

// Left without a computer, provider or model, the run goes to the starter's first ready computer with their defaults.
export const useRunPlay = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:run", ({ playId, input }: { playId: string; input: RunPlayInput }) =>
      api.post<Trail>(`/api/plays/${playId}/run`, { ...input, memory_ids: [], model_options: [] }),
    ),
    onSuccess: (trail) => settleRun(client, trail),
  });
};

export const useStopRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:write", (trailId: string) => api.post<Trail>(`/api/plays/runs/${trailId}/stop`)),
    onSuccess: (trail) => settleRun(client, trail),
  });
};

export const useAnswerRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:write", ({ trailId, answers }: { trailId: string; answers: Record<string, AnswerValue> }) =>
      api.post<Trail>(`/api/plays/runs/${trailId}/answer`, { answers }),
    ),
    onSuccess: (trail) => settleRun(client, trail),
  });
};

// Answers with the run the message went to: the same one, or a new run when its T3 Code thread was deleted.
export const useContinueRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:write", ({ trailId, message }: { trailId: string; message: string }) =>
      api.post<Trail>(`/api/plays/runs/${trailId}/continue`, { message }),
    ),
    onSuccess: (trail) => settleRun(client, trail),
  });
};
