import type { PlayType } from "@/models/Play";

export type PlayQueueStatus = "queued" | "dispatching" | "started" | "skipped" | "didnt_run" | "cancelled";

// Mirrors internal/plays.QueueItem: one match of an auto play's moment, waiting on its person or decided.
export interface PlayQueueItem {
  id: string;
  workspace_id: string;
  project_id: string;
  target_type: PlayType;
  target_id: string;
  play_id: string;
  play_label: string;
  auto_play_id: string;
  person_id: string;
  run_on: "developer" | "tester" | "causer";
  moment: string;
  priority: "high" | "normal" | "low";
  status: PlayQueueStatus;
  // While queued: offline, ticket busy, paused, or "" waiting for a free slot; once decided, why.
  reason: string;
  trail_id: string;
  via: "web" | "mcp";
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
}
