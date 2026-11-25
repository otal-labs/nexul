import { z } from "zod";

import type { TrailState } from "@nexul/client-core/trail";

import type { OptionSetting, PassOver } from "@/models/Pairing";
import type { PlayType } from "@/models/Play";
import type { HarnessQuestion, QuestionAnswer } from "@/models/Question";

// A target's runs come newest first, so a play's latest run is its first one in the list.
export const latestRunOf = (trails: Trail[] | undefined, playId: string): Trail | undefined => trails?.find((t) => t.play_id === playId);

// Mirrors internal/plays.TrailQuestion: the question a run stopped on, with its answer beside it once given.
export interface TrailQuestion extends HarnessQuestion {
  answer?: QuestionAnswer;
  asked_at: string;
}

// Mirrors internal/harness.ActivityKind plus plays.ActivityNote: what one transcript step is; a note is the
// runner's own line (a skipped move, a stop), not harness activity, and a user_message is one the starter wrote in
// the harness itself, its tool naming the harness, and reasoning is the Agent's summary of its own thinking.
export const ACTIVITY_KINDS = ["tool_call", "tool_result", "text", "question", "other", "note", "user_message", "reasoning"] as const;
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

// Mirrors internal/plays.FoundMemory: a memory a run read by itself, by its title now or, once deleted, the one it had.
export interface FoundMemory {
  id: string;
  title: string;
  deleted: boolean;
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
  // The memories it read that it wasn't told to, in read order; sent on one trail to a reader of its project's memories.
  found_memories?: FoundMemory[];
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

// What Continue sends an ended run's own harness thread; the play's instructions never go again (ADR 0128).
export const ContinueTrailFormSchema = z.object({
  message: z.string().trim().min(1, "Write what the agent should do next"),
});

export type ContinueTrailFormData = z.infer<typeof ContinueTrailFormSchema>;

// Mirrors plays.MemoryReadOf without the id: a finished, unfailed memory_get step under any harness's prefix for the tool.
export const isMemoryRead = (step: ActivityEntry): boolean =>
  step.kind === "tool_result" && !step.summary.endsWith(" · failed") && /(^|[^\p{L}\p{N}])memory_get$/u.test(step.tool ?? "");
