import { AutomationRunOutcome } from "@/enums/Automation";
import type { AutomationRun } from "@/models/AutomationRun";
import { chatDayLabel } from "@/utils/ChatDayUtility";

// The server's page: the runs list holds at most this many, so a full list is a window, not every run.
export const RUNS_PAGE = 50;

export interface AutomationRunDay {
  label: string;
  runs: AutomationRun[];
}

// Runs come newest first, so each day is one stretch of the list.
export const automationRunDays = (runs: readonly AutomationRun[], now = new Date()): AutomationRunDay[] =>
  runs.reduce<AutomationRunDay[]>((days, run) => {
    const label = chatDayLabel(run.started_at, now);
    const last = days.at(-1);
    if (last?.label === label) {
      last.runs.push(run);
      return days;
    }
    return [...days, { label, runs: [run] }];
  }, []);

// "Newest 50 runs · 7 failed" for a full page, "12 runs · 3 failed" when the list holds every kept run.
export const automationRunsLine = (runs: readonly AutomationRun[], failedOnly: boolean): string => {
  const shown = runs.length >= RUNS_PAGE ? `Newest ${runs.length}` : `${runs.length}`;
  if (failedOnly) return `${shown} ${runs.length === 1 ? "failure" : "failures"}`;
  const failed = runs.filter((run) => run.outcome !== AutomationRunOutcome.Success).length;
  const total = `${shown} ${runs.length === 1 ? "run" : "runs"}`;
  return failed === 0 ? total : `${total} · ${failed} failed`;
};

export interface AutomationLogLine {
  time?: string | undefined;
  level?: string | undefined;
  message: string;
  fields: string;
}

const textOf = (value: unknown): string => (typeof value === "string" ? value : JSON.stringify(value));

// The SDK writes one JSON object per line ({ts, level, message, ...fields}); anything else stays a raw message.
export const automationLogLine = (raw: string): AutomationLogLine => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { message: raw, fields: "" };
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return { message: raw, fields: "" };
  const { ts, level, message, msg, ...rest } = parsed as Record<string, unknown>;
  const text = message ?? msg;
  if (typeof text !== "string") return { message: raw, fields: "" };
  const date = typeof ts === "string" ? new Date(ts) : undefined;
  return {
    time: date && !Number.isNaN(date.getTime()) ? date.toLocaleTimeString([], { hourCycle: "h23" }) : undefined,
    level: typeof level === "string" ? level : undefined,
    message: text,
    fields: Object.entries(rest)
      .map(([key, value]) => `${key}=${textOf(value)}`)
      .join(" "),
  };
};
