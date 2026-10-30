import { z } from "zod";

import { stepLabel } from "@nexul/client-core/agentSteps";

import type { OptionSetting, PassOver } from "@/models/Pairing";
import type { PlayType } from "@/models/Play";
import type { HarnessQuestion, QuestionAnswer } from "@/models/Question";

// Mirrors internal/plays TrailState: starting precedes the harness accepting, waiting is a turn stopped on a
// question to the user, the last three are terminal.
export const TRAIL_STATES = ["starting", "running", "waiting", "done", "failed", "interrupted"] as const;
export type TrailState = (typeof TRAIL_STATES)[number];

// A target's runs come newest first, so a play's latest run is its first one in the list.
export const latestRunOf = (trails: Trail[] | undefined, playId: string): Trail | undefined => trails?.find((t) => t.play_id === playId);

export const isTrailActive = (state: TrailState): boolean => state === "starting" || state === "running" || state === "waiting";

// Mirrors internal/plays.TrailQuestion: the question a run stopped on, with its answer beside it once given.
export interface TrailQuestion extends HarnessQuestion {
  answer?: QuestionAnswer;
  asked_at: string;
}

// Mirrors internal/harness.ActivityKind plus plays.ActivityNote: what one transcript step is; a note is the
// runner's own line (a skipped move, a stop), not harness activity, and a user_message is one the starter wrote in
// the harness itself, its tool naming the harness.
export const ACTIVITY_KINDS = ["tool_call", "tool_result", "text", "question", "other", "note", "user_message"] as const;
export type ActivityKind = (typeof ACTIVITY_KINDS)[number];

// Mirrors internal/plays.ActivityEntry, one step of a trail's transcript; detail is JSON ({input, result}) or plain text.
export interface ActivityEntry {
  kind: ActivityKind;
  call_id?: string;
  tool?: string;
  summary: string;
  detail?: string;
  at: string;
}

// Mirrors internal/plays.Trail, the persisted record of one play run.
export interface Trail {
  id: string;
  workspace_id: string;
  play_id: string;
  play_label: string;
  target_type: PlayType;
  target_id: string;
  project_id: string;
  conversation_id: string;
  starter_id: string;
  via: "web" | "mcp" | "automation";
  selected_memory_ids: string[];
  custom_instructions: string;
  computer_id: string;
  provider: string;
  model: string;
  harness_session_id: string;
  state: TrailState;
  started_at: string;
  ended_at: string | null;
  last_error: string;
  // Mirrors plays.Trail.FailureReason: the harness refusal's reason (pairing's NotConfiguredReason), "" otherwise.
  failure_reason: string;
  // The computers it could not land on, before the one it ran on or when it ran on none (ADR 0165); its starter alone gets them.
  passed_over?: PassOver[];
  reply_message_id: string;
  activity: ActivityEntry[];
  question?: TrailQuestion | null;
  // The play instructions version the run started with; absent on runs from before history.
  instructions_version?: number;
}

// Mirrors internal/plays.PlayRun: a run on a play's runs list with what it ran on (a ticket's key and title, or a doc's,
// stack's or project's name) and the auto play that started it, "" for one a person or an automation started.
export interface PlayRun extends Trail {
  target_key: string;
  target_title: string;
  auto_play_id: string;
}

export interface PlayRunsPage {
  items: PlayRun[];
  total: number;
}

// Mirrors internal/plays.RunFrame, the live-hub payload on topic play.run.
export interface RunFrame {
  trail_id: string;
  play_id: string;
  target_type: PlayType;
  target_id: string;
  state: TrailState;
  activity: ActivityEntry | null;
  question?: TrailQuestion | null;
  ended_at: string | null;
  last_error: string;
}

// Where a run frame's run sits, for the followers whose views are keyed by project or workspace.
export interface RunPlace {
  project_id: string;
  workspace_id: string;
}

// What the caller last picked for one play in one project, read from their latest trail. An empty
// computer_id means never run; the pre-selection falls back to the resolved target instead.
export interface LatestChoices {
  memory_ids: string[];
  computer_id: string;
  provider: string;
  model: string;
  model_options?: OptionSetting[];
}

export interface RunPlayInput {
  target_type: PlayType;
  target_id: string;
  memory_ids: string[];
  custom_instructions: string;
  // Sent when the person picks where this run goes; the server remembers the T3 project as the project's checkout on
  // that computer (ADR 0165). Left out, the run goes to their first ready computer.
  computer_id?: string;
  harness_project_id?: string;
  provider: string;
  model: string;
  model_options: OptionSetting[];
}

const STATE_LABELS: Record<TrailState, string> = {
  starting: "Starting…",
  running: "Running…",
  waiting: "Waiting for your answer",
  done: "Done",
  failed: "Failed",
  interrupted: "Interrupted",
};

// The row's summary stays one line even for a multi-line or long harness error; TrailDetailBody shows the
// untruncated trail.last_error in its own section.
const SUMMARY_ERROR_MAX_LENGTH = 80;

const truncateSummaryError = (text: string): string => {
  const firstLine = text.split("\n")[0] ?? "";
  if (firstLine.length <= SUMMARY_ERROR_MAX_LENGTH) return firstLine;
  return `${firstLine.slice(0, SUMMARY_ERROR_MAX_LENGTH)}…`;
};

const lastIndexOfCall = (activity: ActivityEntry[], callId: string): number => {
  for (let i = activity.length - 1; i >= 0; i--) {
    if (activity[i]?.call_id === callId) return i;
  }
  return -1;
};

// The live frame carries only the newest step; it replaces the step sharing its call id, else joins the end.
export const mergeLiveStep = (activity: ActivityEntry[], live: ActivityEntry | null): ActivityEntry[] => {
  if (live === null) return activity;
  if (live.call_id) {
    const i = lastIndexOfCall(activity, live.call_id);
    if (i !== -1) return activity.map((e, j) => (j === i ? live : e));
  }
  const last = activity[activity.length - 1];
  if (last && last.at === live.at && last.summary === live.summary) return activity;
  return [...activity, live];
};

// Folds the steps a run pushed while the page watched onto the persisted list: a call id replaces its earlier
// entry (a result never steps back to its call), anything else is skipped when the list already has it.
export const mergeLiveSteps = (activity: ActivityEntry[], live: ActivityEntry[]): ActivityEntry[] =>
  live.reduce<ActivityEntry[]>((acc, step) => {
    if (step.call_id) {
      const i = lastIndexOfCall(acc, step.call_id);
      if (i === -1) return [...acc, step];
      if (acc[i]?.kind === "tool_result" && step.kind === "tool_call") return acc;
      return acc.map((e, j) => (j === i ? step : e));
    }
    if (acc.some((e) => e.at === step.at && e.summary === step.summary)) return acc;
    return [...acc, step];
  }, activity);

// What Continue sends an ended run's own harness thread; the play's instructions never go again (ADR 0128).
export const ContinueTrailFormSchema = z.object({
  message: z.string().trim().min(1, "Write what the agent should do next"),
});

export type ContinueTrailFormData = z.infer<typeof ContinueTrailFormSchema>;

// One line for a trail row: the latest step while it runs, the reason once it ended badly.
export const trailSummary = (state: TrailState, lastError: string, activity: ActivityEntry | null | undefined): string => {
  if (state === "running" && activity && activity.kind !== "user_message") return stepLabel(activity);
  if ((state === "failed" || state === "interrupted") && lastError !== "") {
    return `${STATE_LABELS[state]} · ${truncateSummaryError(lastError)}`;
  }
  return STATE_LABELS[state];
};
