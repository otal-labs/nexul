import type { PlayStage } from "@nexul/client-core/plays";


// Mirrors internal/plays/autoplay.go's Schedule: when a schedule auto play sweeps and how much one sweep may start.
export interface AutoPlaySchedule {
  cron: string;
  timezone?: string;
  // "HH:MM" in the timezone; absent for no stop.
  stop_at?: string;
  max_runs?: number;
}

export const DEFAULT_SWEEP_RUNS = 10;
export const MAX_SWEEP_RUNS = 100;

export interface SweepCounts {
  finished: number;
  asked: number;
  failed: number;
  skipped: number;
  didnt_run: number;
}

// Mirrors internal/plays/sweep.go: one firing of a schedule, its tickets queued in order, closed once nothing waits or works.
export interface Sweep {
  id: string;
  workspace_id: string;
  auto_play_id: string;
  play_id: string;
  play_label: string;
  fired_at: string;
  start_by: string | null;
  matched: number;
  over_cap: number;
  counts: SweepCounts;
  finished_at: string | null;
}

export type SweepOutcome = "" | "running" | "finished" | "asked" | "failed";

export interface SweepLine {
  queue_item_id: string;
  ticket_id: string;
  key: string;
  title: string;
  status: "queued" | "dispatching" | "started" | "skipped" | "didnt_run" | "cancelled";
  reason: string;
  trail_id: string;
  outcome: SweepOutcome;
  question: string;
  stage: PlayStage | "";
  bugs: { key: string; title: string; done: boolean }[];
}

export interface SweepDigest extends Sweep {
  lines: SweepLine[];
}

export const validZone = (zone: string) => {
  try {
    new Intl.DateTimeFormat("en", { timeZone: zone });
    return true;
  } catch {
    return false;
  }
};

// Whether the text has the five fields a schedule needs; the server checks the rest.
export const looksLikeCron = (cron: string) => cron.trim().split(/\s+/).length === 5;

// "4 finished, 1 asked, 1 skipped": only the counts that happened.
export const countsPhrase = (counts: SweepCounts): string =>
  [
    [counts.finished, "finished"],
    [counts.asked, "asked"],
    [counts.failed, "failed"],
    [counts.skipped, "skipped"],
    [counts.didnt_run, "didn't run"],
  ]
    .filter(([n]) => Number(n) > 0)
    .map(([n, word]) => `${n} ${word}`)
    .join(", ");

// A time inside a sentence: "today at 23:00" rather than the list's "Today at 23:00".
export const inSentence = (when: string) => when.replace(/^(Today|Yesterday)/, (word) => word.toLowerCase());
