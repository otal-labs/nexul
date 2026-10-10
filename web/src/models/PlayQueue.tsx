import type { PlayType } from "@/models/Play";

export type PlayQueueStatus = "queued" | "dispatching" | "started" | "skipped" | "didnt_run" | "cancelled";

// Mirrors internal/plays.QueueItem: an auto play's match, or an automation's runPlay call, waiting on its person or decided.
export interface PlayQueueItem {
  id: string;
  workspace_id: string;
  project_id: string;
  target_type: PlayType;
  target_id: string;
  play_id: string;
  play_label: string;
  // Empty for a run an automation queued, which names automation_id instead.
  auto_play_id: string;
  automation_id: string;
  person_id: string;
  run_on: "developer" | "tester" | "causer";
  moment: string;
  priority: "high" | "normal" | "low";
  status: PlayQueueStatus;
  // While queued: offline, ticket busy, paused, or "" waiting for a free slot; once decided, why.
  reason: string;
  trail_id: string;
  via: "web" | "mcp" | "automation";
  queued_at: string;
  decided_at: string | null;
  not_before: string;
}

// Mirrors internal/plays.Queue: a ticket's or doc's auto runs, newest first, and whether the daily cap paused them.
export interface PlayQueue {
  items: PlayQueueItem[];
  paused: boolean;
  auto_runs: number;
  daily_cap: number;
  // When the oldest counted run leaves the rolling day; null unless paused.
  paused_until: string | null;
}
