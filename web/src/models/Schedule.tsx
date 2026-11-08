import type { PlayStage } from "@/models/Play";

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

const DAY_NAMES = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
const DAY_ABBREVIATIONS = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];

const isNumber = (field: string) => /^\d{1,2}$/.test(field);
const twoDigits = (n: string) => n.padStart(2, "0");

// A zone as a person names it: "Europe/London" reads "London", "America/New_York" reads "New York".
export const zoneName = (zone: string) => (zone.split("/").at(-1) ?? zone).replaceAll("_", " ");

export const validZone = (zone: string) => {
  try {
    new Intl.DateTimeFormat("en", { timeZone: zone });
    return true;
  } catch {
    return false;
  }
};

const dayNumber = (field: string) => {
  const index = DAY_ABBREVIATIONS.indexOf(field.toLowerCase());
  if (index >= 0) return index;
  return isNumber(field) ? Number(field) % 7 : -1;
};

// The day-of-week field in words: "Every day", "Weekdays", "Weekends", "Mondays and Thursdays"; "" when it is no plain list.
const daysPhrase = (dow: string): string => {
  if (dow === "*") return "Every day";
  if (["1-5", "mon-fri"].includes(dow.toLowerCase())) return "Weekdays";
  const days = dow.split(",").map(dayNumber);
  if (days.some((day) => day < 0)) return "";
  if (days.length === 2 && days.includes(0) && days.includes(6)) return "Weekends";
  const names = days.map((day) => `${DAY_NAMES[day]}s`);
  return names.length === 1 ? (names[0] ?? "") : `${names.slice(0, -1).join(", ")} and ${names.at(-1)}`;
};

// The cron in plain words for the common shapes, "Every day at 23:00, London"; anything else reads as a custom schedule.
export const describeSchedule = (cron: string, zone: string): string => {
  const fields = cron.trim().split(/\s+/);
  if (fields.length !== 5) return "";
  const [minute = "", hour = "", dom = "", month = "", dow = ""] = fields;
  const place = zoneName(zone || "UTC");
  if (!isNumber(minute) || month !== "*") return `A custom schedule, ${place}`;
  if (hour === "*" && dom === "*" && dow === "*") return `Every hour at :${twoDigits(minute)}, ${place}`;
  if (!isNumber(hour)) return `A custom schedule, ${place}`;
  const time = `${twoDigits(hour)}:${twoDigits(minute)}`;
  if (isNumber(dom) && dow === "*") return `On day ${dom} of each month at ${time}, ${place}`;
  const days = dom === "*" ? daysPhrase(dow) : "";
  if (days === "") return `A custom schedule, ${place}`;
  return `${days} at ${time}, ${place}`;
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
