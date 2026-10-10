import type { PlayQueue, PlayQueueItem } from "@/models/PlayQueue";

// One auto run as the queue routes return it, queued for alice on ticket t-1 unless overridden.
export const queueItem = (overrides: Partial<PlayQueueItem> = {}): PlayQueueItem => ({
  id: "q-1",
  workspace_id: "ws-1",
  project_id: "p-1",
  target_type: "ticket",
  target_id: "t-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  auto_play_id: "ap-1",
  automation_id: "",
  person_id: "u-alice",
  run_on: "developer",
  moment: "ticket.unblocked",
  priority: "normal",
  status: "queued",
  reason: "",
  trail_id: "",
  via: "web",
  queued_at: "2026-10-10T09:00:00Z",
  decided_at: null,
  not_before: "2026-10-10T09:00:00Z",
  ...overrides,
});

export const playQueue = (overrides: Partial<PlayQueue> = {}): PlayQueue => ({
  items: [],
  paused: false,
  auto_runs: 0,
  daily_cap: 5,
  paused_until: null,
  ...overrides,
});
