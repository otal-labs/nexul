import type { PlayType } from "@/models/Play";
import type { HarnessQuestion, QuestionAnswer } from "@/models/Question";

// Mirrors internal/plays TrailState: starting precedes the harness accepting, waiting is a turn stopped on a
// question to the user, the last three are terminal.
export const TRAIL_STATES = ["starting", "running", "waiting", "done", "failed", "interrupted"] as const;
export type TrailState = (typeof TRAIL_STATES)[number];

export const isTrailActive = (state: TrailState): boolean => state === "starting" || state === "running" || state === "waiting";

// Mirrors internal/plays.TrailQuestion: the question a run stopped on, with its answer beside it once given.
export interface TrailQuestion extends HarnessQuestion {
  answer?: QuestionAnswer;
  asked_at: string;
}

// Mirrors internal/harness.ActivityKind plus plays.ActivityNote: what one transcript step is; a note is the
// runner's own line (a skipped move, a stop), not harness activity.
export const ACTIVITY_KINDS = ["tool_call", "tool_result", "text", "question", "other", "note"] as const;
export type ActivityKind = (typeof ACTIVITY_KINDS)[number];

// The harness's built-in tools whose row reads the command or the path alone, as the harness itself labels them.
const COMMAND_TOOLS = new Set(["Bash", "Shell"]);
const FILE_TOOLS = new Set(["Edit", "Write", "MultiEdit", "NotebookEdit"]);

export const isCommandTool = (tool: string | undefined): boolean => tool !== undefined && COMMAND_TOOLS.has(tool);
export const isFileTool = (tool: string | undefined): boolean => tool !== undefined && FILE_TOOLS.has(tool);

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
  via: "web" | "mcp";
  selected_memory_ids: string[];
  custom_instructions: string;
  move_to_status_id: string;
  computer_id: string;
  provider: string;
  model: string;
  harness_session_id: string;
  state: TrailState;
  started_at: string;
  ended_at: string | null;
  last_error: string;
  reply_message_id: string;
  activity: ActivityEntry[];
  question?: TrailQuestion | null;
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

// What the caller last picked for one play in one project, read from their latest trail. An empty
// computer_id means never run; the pre-selection falls back to the resolved target instead.
export interface LatestChoices {
  memory_ids: string[];
  move_to_status_id: string;
  computer_id: string;
  provider: string;
  model: string;
}

export interface RunPlayInput {
  target_type: PlayType;
  target_id: string;
  memory_ids: string[];
  custom_instructions: string;
  move_to_status_id: string;
  computer_id: string;
  provider: string;
  model: string;
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

// One line for a step: `tool: args` for a tool call, the command alone for a command, the summary alone for text,
// notes, and legacy lines.
export const stepLabel = (entry: ActivityEntry): string => {
  if (!entry.tool || isCommandTool(entry.tool)) return entry.summary;
  return `${entry.tool}: ${entry.summary}`;
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

// One line for a trail row: the latest step while it runs, the reason once it ended badly.
export const trailSummary = (state: TrailState, lastError: string, activity: ActivityEntry | null | undefined): string => {
  if (state === "running" && activity) return stepLabel(activity);
  if ((state === "failed" || state === "interrupted") && lastError !== "") {
    return `${STATE_LABELS[state]} · ${truncateSummaryError(lastError)}`;
  }
  return STATE_LABELS[state];
};
