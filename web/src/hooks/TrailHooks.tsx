import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { toast } from "sonner";
import { useShallow } from "zustand/react/shallow";

import { api, errorMessage } from "@/api/client";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { conversationPlayTarget, type Conversation } from "@/models/Chat";
import type { PlayType } from "@/models/Play";
import type { QuestionAnswers } from "@/models/Question";
import { DECISIONS_CHECK_PLAY_ID, isTrailActive, mergeLiveSteps, type ActivityEntry, type LatestChoices, type RunPlayInput, type Trail, type TrailQuestion, type TrailState } from "@/models/Trail";
import { targetKey, usePlayRunStore, type PlayRunStore } from "@/stores/playRunStore";
import { threadTrailBlocks, type ThreadTrailBlocks } from "@/utils/ThreadTrailUtility";

export const getTrailsKey = "getTrails";
export const getTrailKey = "getTrail";
export const getActiveTrailsKey = "getActiveTrails";
export const getLatestChoicesKey = "getLatestChoices";

export const useFetchTrails = (targetType: PlayType, targetId: string) =>
  useQuery({
    queryKey: [getTrailsKey, targetType, targetId],
    queryFn: async () =>
      (await api.get<Trail[]>("/api/plays/runs", { params: { target_type: targetType, target_id: targetId } })).data,
    enabled: targetId !== "",
  });

export const useFetchTrail = (trailId: string | null) =>
  useQuery({
    queryKey: [getTrailKey, trailId],
    queryFn: async () => (await api.get<Trail>(`/api/plays/runs/${trailId}`)).data,
    enabled: trailId !== null && trailId !== "",
  });

export const useFetchLatestChoices = (playId: string, projectId: string) =>
  useQuery({
    queryKey: [getLatestChoicesKey, playId, projectId],
    queryFn: async () =>
      (await api.get<LatestChoices>("/api/plays/latest-choices", { params: { play_id: playId, project_id: projectId } })).data,
    enabled: playId !== "" && projectId !== "",
  });

// Target id to trail id; waiting holds the subset stopped on a question to the user, started each run's start time.
interface ActiveTargets {
  active: Record<string, string>;
  waiting: Record<string, string>;
  started: Record<string, string>;
}

// One request per project and target type: every row of a board or list shares the key.
const useFetchActiveTargets = (targetType: PlayType, projectId: string | undefined, ids: string[]) =>
  useQuery({
    queryKey: [getActiveTrailsKey, targetType, projectId, ids],
    queryFn: async (): Promise<ActiveTargets> => {
      const { data } = await api.get<Partial<ActiveTargets>>("/api/plays/runs/active", { params: { target_type: targetType, target_ids: ids.join(",") } });
      return { active: data.active ?? {}, waiting: data.waiting ?? {}, started: data.started ?? {} };
    },
    enabled: !!projectId && ids.length > 0,
  });

export const useFetchActiveTrails = (projectId: string | undefined) => {
  const { data: tickets } = useFetchTicketsByProject(projectId);
  return useFetchActiveTargets("ticket", projectId, tickets?.map((t) => t.id) ?? []);
};

export const useFetchActiveDocTrails = (projectId: string) => {
  const { data: docs } = useFetchDocsByProject(projectId);
  return useFetchActiveTargets("doc", projectId, docs?.map((d) => d.id) ?? []);
};

// Errors render inside the run dialog, so no toast here.
export const useRunPlay = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ playId, input }: { playId: string; input: RunPlayInput }) =>
      (await api.post<Trail>(`/api/plays/${playId}/run`, input)).data,
    onSuccess: async (trail) => {
      client.setQueryData<Trail[]>([getTrailsKey, trail.target_type, trail.target_id], (old) => [
        trail,
        ...(old ?? []).filter((t) => t.id !== trail.id),
      ]);
      await client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id] });
      await client.invalidateQueries({ queryKey: [getActiveTrailsKey] });
      await client.invalidateQueries({ queryKey: [getLatestChoicesKey, trail.play_id, trail.project_id] });
      toast.success(`${trail.play_label} started`);
    },
  });
};

export const useRunDecisionsCheck = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (ticketId: string) => (await api.post<Trail>("/api/plays/decisions-check", { ticket_id: ticketId })).data,
    onSuccess: async (trail) => {
      client.setQueryData<Trail[]>([getTrailsKey, trail.target_type, trail.target_id], (old) => [
        trail,
        ...(old ?? []).filter((t) => t.id !== trail.id),
      ]);
      await client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id] });
      await client.invalidateQueries({ queryKey: [getActiveTrailsKey] });
      toast.success("Decisions check started");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The ticket's latest decisions check when it failed, so the page can offer to run it again; trails come newest first.
export const useMissedDecisionsCheck = (ticketId: string): Trail | undefined => {
  const { data: trails } = useFetchTrails("ticket", ticketId);
  const frames = usePlayRunStore((s) => s.frames);
  const latest = trails?.find((t) => t.play_id === DECISIONS_CHECK_PLAY_ID);
  if (!latest || (frames[latest.id]?.state ?? latest.state) !== "failed") return undefined;
  return latest;
};

export const useStopTrail = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (trailId: string) => (await api.post<Trail>(`/api/plays/runs/${trailId}/stop`)).data,
    onSuccess: async (trail) => {
      client.setQueryData<Trail[]>([getTrailsKey, trail.target_type, trail.target_id], (old) =>
        (old ?? []).map((t) => (t.id === trail.id ? trail : t)),
      );
      await client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id] });
      await client.invalidateQueries({ queryKey: [getTrailKey, trail.id] });
      await client.invalidateQueries({ queryKey: [getActiveTrailsKey] });
      toast.success("Run stopped");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The answer continues the run on the same trail; the live frame then flips the state before the refetch lands.
export const useAnswerTrail = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ trailId, answers }: { trailId: string; answers: QuestionAnswers }) =>
      (await api.post<Trail>(`/api/plays/runs/${trailId}/answer`, { answers })).data,
    onSuccess: async (trail) => {
      client.setQueryData<Trail>([getTrailKey, trail.id], trail);
      await client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id] });
      await client.invalidateQueries({ queryKey: [getTrailKey, trail.id] });
      toast.success("Answer sent");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Continue answers with the trail the message went to: the same run, or a new one when its harness thread was deleted.
export const useContinueTrail = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ trailId, message }: { trailId: string; message: string }) =>
      (await api.post<Trail>(`/api/plays/runs/${trailId}/continue`, { message })).data,
    onSuccess: async (trail, { trailId }) => {
      client.setQueryData<Trail>([getTrailKey, trail.id], trail);
      await client.invalidateQueries({ queryKey: [getTrailsKey, trail.target_type, trail.target_id] });
      await client.invalidateQueries({ queryKey: [getTrailKey, trailId] });
      await client.invalidateQueries({ queryKey: [getActiveTrailsKey] });
      if (trail.id !== trailId) {
        toast.info("Its thread was deleted in T3 Code, so the play started again as a new run");
        return;
      }
      toast.success("Sent to the run");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A trail's state as the page should show it: the live frame wins over the fetched row until the refetch lands.
export const useLiveTrailState = (trail: Trail) =>
  usePlayRunStore((s) => s.frames[trail.id]?.state ?? trail.state);

export const useLiveTrailActivity = (trail: Trail) =>
  usePlayRunStore((s) => s.frames[trail.id]?.activity ?? null);

export const useLiveTrailQuestion = (trail: Trail): TrailQuestion | null =>
  usePlayRunStore((s) => s.frames[trail.id]?.question ?? trail.question ?? null);

// A stable empty list, so a trail with no live steps never hands the selector a fresh array per render.
const NO_STEPS: ActivityEntry[] = [];

// Every step of a trail as the page should show it: the fetched list plus the steps the frames carried since.
export const useLiveTrailSteps = (trail: Trail): ActivityEntry[] => {
  const live = usePlayRunStore((s) => s.steps[trail.id] ?? NO_STEPS);
  return useMemo(() => mergeLiveSteps(trail.activity, live), [trail.activity, live]);
};

// The runs a thread shows inline: each run's turns placed above the message it led to, plus the running one.
export const useThreadTrailBlocks = (conversation: Conversation): ThreadTrailBlocks => {
  const target = conversationPlayTarget(conversation);
  const { data: trails } = useFetchTrails(target?.type ?? "ticket", target?.id ?? "");
  const frames = usePlayRunStore((s) => s.frames);
  const steps = usePlayRunStore((s) => s.steps);
  return useMemo(
    () =>
      threadTrailBlocks(
        (trails ?? [])
          .filter((t) => t.conversation_id === conversation.id)
          .map((t) => ({
            trail: t,
            steps: mergeLiveSteps(t.activity, steps[t.id] ?? NO_STEPS),
            state: frames[t.id]?.state ?? t.state,
            question: frames[t.id]?.question ?? t.question ?? null,
          })),
      ),
    [trails, frames, steps, conversation.id],
  );
};

// The one trail occupying the target, if any; drives the disabled and Stop states of every play button on it.
export const useActiveTrail = (targetType: PlayType, targetId: string): Trail | undefined => {
  const { data: trails } = useFetchTrails(targetType, targetId);
  const frames = usePlayRunStore((s) => s.frames);
  return trails?.find((t) => isTrailActive(frames[t.id]?.state ?? t.state));
};

// Joins the on-load batch answer with live frames so a start, a question, or an end flips it without a refetch.
const runStateOf = (s: PlayRunStore, targetType: PlayType, targetId: string, known: ActiveTargets | undefined): TrailState | undefined => {
  const trailId = s.activeByTarget[targetKey(targetType, targetId)] ?? known?.active[targetId];
  if (!trailId) return undefined;
  const state = s.frames[trailId]?.state ?? (known?.waiting[targetId] === trailId ? "waiting" : "running");
  return isTrailActive(state) ? state : undefined;
};

const useRunState = (targetType: PlayType, targetId: string, known: ActiveTargets | undefined): TrailState | undefined =>
  usePlayRunStore((s) => runStateOf(s, targetType, targetId, known));

// The board card's question: no run, a run going, or a run waiting on an answer.
export const useTicketRunState = (projectId: string, ticketId: string): TrailState | undefined =>
  useRunState("ticket", ticketId, useFetchActiveTrails(projectId).data);

// When the ticket's run started, for the card's timer; undefined until the batch answer carries it.
export const useTicketRunStartedAt = (projectId: string, ticketId: string): string | undefined =>
  useFetchActiveTrails(projectId).data?.started[ticketId];

// A swimlane header's question: how many of its tickets have a run going, and how many wait on an answer.
export const useTicketRunCounts = (projectId: string | undefined, ticketIds: string[]) => {
  const known = useFetchActiveTrails(projectId).data;
  return usePlayRunStore(
    useShallow((s) => {
      const states = ticketIds.map((id) => runStateOf(s, "ticket", id, known));
      return {
        running: states.filter((state) => state !== undefined && state !== "waiting").length,
        waiting: states.filter((state) => state === "waiting").length,
      };
    }),
  );
};

// The doc list row's question.
export const useDocRunState = (projectId: string, docId: string): TrailState | undefined =>
  useRunState("doc", docId, useFetchActiveDocTrails(projectId).data);
