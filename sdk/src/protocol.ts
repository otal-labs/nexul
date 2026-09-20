// Mirrors internal/automations/protocol.go's Frame exactly (ADR 0046): the flat
// JSON wire message both sides speak. Keep this file in lockstep with that
// one by hand — it's small and rarely changes, so generating it isn't worth
// the machinery the event catalog needed.

export type FrameType =
  | "announce"
  | "hello"
  | "event"
  | "run_started"
  | "run_log"
  | "run_finished"
  | "run_crashed";

export type RunOutcome = "success" | "failure";

export interface Frame {
  type: FrameType;
  name?: string;
  description?: string;
  subscriptions?: string[];
  config_schema?: unknown;
  config_values?: unknown;
  secrets?: Record<string, string>;
  run_id?: string;
  event_id?: string;
  topic?: string;
  payload?: unknown;
  log?: string;
  outcome?: RunOutcome;
  error?: string;
}

export function announceFrame(name: string, description: string, subscriptions: string[], configSchema: unknown): Frame {
  return { type: "announce", name, description, subscriptions, config_schema: configSchema };
}

export function runStartedFrame(runId: string): Frame {
  return { type: "run_started", run_id: runId };
}

export function runLogFrame(runId: string, log: string): Frame {
  return { type: "run_log", run_id: runId, log };
}

export function runFinishedFrame(runId: string, outcome: RunOutcome): Frame {
  return { type: "run_finished", run_id: runId, outcome };
}

export function runCrashedFrame(runId: string, error: string): Frame {
  return { type: "run_crashed", run_id: runId, error };
}
