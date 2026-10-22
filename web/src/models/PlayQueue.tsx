import type { PassOver } from "@/models/Pairing";
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
  run_on: "developer" | "tester" | "causer" | "computer";
  moment: string;
  priority: "high" | "normal" | "low";
  status: PlayQueueStatus;
  // While queued: ticket busy, paused, why none of its person's computers can take it (unpaired, expired_token,
  // setup_required, offline, no_computer_ready), or "" until first tried; once decided, why.
  reason: string;
  trail_id: string;
  via: "web" | "mcp" | "automation";
  queued_at: string;
  decided_at: string | null;
  not_before: string;
  // The computer its auto play names for "computer"; "" when it goes to its person's first ready computer.
  computer_id: string;
  // For a stack moment, the service that went down or came back and its outage; "" otherwise.
  service: string;
  outage_id: string;
  // While queued, to its person alone: each of their computers and why it can't take the run.
  waiting_on?: PassOver[];
}

// Mirrors internal/plays.Queue: a ticket's, doc's or stack's auto runs, newest first, and whether the daily cap paused them.
export interface PlayQueue {
  items: PlayQueueItem[];
  paused: boolean;
  auto_runs: number;
  daily_cap: number;
  // When the oldest counted run leaves the rolling day; null unless paused.
  paused_until: string | null;
}
