import { z } from "zod";

import type { OptionSetting } from "@/models/Pairing";
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
// runner's own line (a skipped move, a stop), not harness activity, and a user_message is one the starter wrote in
// the harness itself, its tool naming the harness.
export const ACTIVITY_KINDS = ["tool_call", "tool_result", "text", "question", "other", "note", "user_message"] as const;
export type ActivityKind = (typeof ACTIVITY_KINDS)[number];

// The harness's built-in tools whose row reads the command or the path alone, as the harness itself labels them.
const COMMAND_TOOLS = new Set(["Bash", "Shell"]);
const FILE_CHANGE_TOOLS = new Set(["Edit", "Write", "MultiEdit", "NotebookEdit"]);
const FILE_READ_TOOLS = new Set(["Read", "Glob", "Grep", "LS", "NotebookRead"]);

export const isCommandTool = (tool: string | undefined): boolean => tool !== undefined && COMMAND_TOOLS.has(tool);
export const isFileReadTool = (tool: string | undefined): boolean => tool !== undefined && FILE_READ_TOOLS.has(tool);
export const isFileTool = (tool: string | undefined): boolean => tool !== undefined && (FILE_CHANGE_TOOLS.has(tool) || FILE_READ_TOOLS.has(tool));

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
  computer_id: string;
  // Sent when the person picks where to run; the server saves it with computer_id as their project link (ADR 0143).
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

// Mirrors the " · failed" the harness appends to a failed tool's summary; the row shows failure on its icon instead.
export const FAILED_SUFFIX = " · failed";

export const isFailedStep = (entry: ActivityEntry): boolean => entry.kind === "tool_result" && entry.summary.endsWith(FAILED_SUFFIX);

const SHELL_WRAPPER = /^(?:\S*\/)?(?:bash|sh|zsh) +-l?c +/;

// The command a `/bin/bash -lc "…"` style wrapper runs, outer quotes off; a summary cut short keeps its open quote's text.
export const unwrapShellCommand = (command: string): string => {
  const match = SHELL_WRAPPER.exec(command);
  if (!match) return command;
  const inner = command.slice(match[0].length).trim();
  const quote = inner.charAt(0);
  if (quote !== '"' && quote !== "'") return inner;
  const closed = inner.length > 1 && inner.endsWith(quote);
  return (closed ? inner.slice(1, -1) : inner.slice(1)).trim();
};

const MCP_TOOL_NAME = /^mcp__(.+?)__(.+)$/i;
const SERVER_SEPARATOR = " · ";

// The server and tool of an MCP call: Claude names it mcp__<server>__<tool>, Codex arrives titled "<server> · <tool>".
const mcpParts = (tool: string): { server: string; name: string } | null => {
  const match = MCP_TOOL_NAME.exec(tool);
  if (match?.[1] && match[2]) return { server: match[1], name: match[2] };
  const at = tool.indexOf(SERVER_SEPARATOR);
  if (at <= 0) return null;
  return { server: tool.slice(0, at), name: tool.slice(at + SERVER_SEPARATOR.length) };
};

const READ_TARGET_KEYS = ["file_path", "notebook_path", "pattern", "path"];

// The path or pattern a read-type tool's JSON arguments name; null when the summary is not whole JSON (a cut-short preview).
const readTarget = (summary: string): string | null => {
  try {
    const args: unknown = JSON.parse(summary);
    if (typeof args !== "object" || args === null) return null;
    const values = args as Record<string, unknown>;
    const key = READ_TARGET_KEYS.find((k) => typeof values[k] === "string" && values[k] !== "");
    return key === undefined ? null : (values[key] as string);
  } catch {
    return null;
  }
};

export const isMcpTool = (tool: string | undefined): boolean => tool !== undefined && mcpParts(tool) !== null;

// One line for a step: `Server · tool` for an MCP call, the command alone out of its shell wrapper, `tool: args` for
// any other tool with a file read's path or pattern for its arguments, the tool alone when its summary only repeats it, the summary alone for text, notes, and legacy lines.
export const stepLabel = (entry: ActivityEntry): string => {
  const summary = entry.summary.endsWith(FAILED_SUFFIX) ? entry.summary.slice(0, -FAILED_SUFFIX.length) : entry.summary;
  if (!entry.tool) return summary;
  if (isCommandTool(entry.tool)) return unwrapShellCommand(summary);
  const mcp = mcpParts(entry.tool);
  if (mcp) return `${mcp.server.charAt(0).toUpperCase()}${mcp.server.slice(1)}${SERVER_SEPARATOR}${mcp.name}`;
  if (summary === "" || summary === entry.tool) return entry.tool;
  const target = isFileReadTool(entry.tool) ? readTarget(summary) : null;
  return `${entry.tool}: ${target ?? summary}`;
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
