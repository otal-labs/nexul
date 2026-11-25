import type { AgentStep } from "@nexul/client-core/agentSteps";
import type { AnswerValue, QuestionItem } from "@nexul/client-core/question";
import type { TrailState } from "@nexul/client-core/trail";

import type { PlayType } from "@/models/Play";

// Mirrors internal/plays.TrailQuestion: the question a run stopped on, with its answer beside it once given.
export interface TrailQuestion {
  request_id: string;
  questions: QuestionItem[];
  answer?: { answers: Record<string, AnswerValue> };
  asked_at: string;
}

// Mirrors internal/plays.Trail, the persisted record of one play run, as the phone reads it.
export interface Trail {
  id: string;
  workspace_id: string;
  play_id: string;
  play_label: string;
  target_type: PlayType;
  target_id: string;
  project_id: string;
  starter_id: string;
  custom_instructions: string;
  state: TrailState;
  started_at: string;
  ended_at: string | null;
  last_error: string;
  activity: AgentStep[];
  question?: TrailQuestion | null;
}

// Mirrors internal/plays.PlayRun: a run on a play's runs list with what it ran on, and the auto play that started it.
export interface PlayRun extends Trail {
  target_key: string;
  target_title: string;
  auto_play_id: string;
}

export interface PlayRunsPage {
  items: PlayRun[];
  total: number;
}

// Mirrors internal/plays.RunFrame, the live hub's play.run payload: the run's state and its newest step.
export interface RunFrame {
  trail_id: string;
  play_id: string;
  target_type: PlayType;
  target_id: string;
  state: TrailState;
  activity: AgentStep | null;
  question: TrailQuestion | null;
  ended_at: string | null;
  last_error: string;
  starter_id: string;
  project_id: string;
  workspace_id: string;
}

// The web's state hues: amber while it works, blue while it asks, green done, red when it failed or was stopped.
const STATE_DOT: Record<TrailState, string> = {
  starting: "bg-warning-mark",
  running: "bg-warning-mark",
  waiting: "bg-info",
  done: "bg-success",
  failed: "bg-destructive",
  interrupted: "bg-destructive",
};

export const trailStateDot = (state: TrailState): string => STATE_DOT[state];

// The frame's fields that move a run, laid over a cached copy; its newest step joins the steps the copy holds.
export const applyRunFrame = <T extends Trail>(trail: T, frame: RunFrame, steps: AgentStep[]): T => ({
  ...trail,
  state: frame.state,
  question: frame.question,
  ended_at: frame.ended_at,
  last_error: frame.last_error,
  activity: steps,
});
