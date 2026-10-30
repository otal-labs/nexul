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
  return `${shown} ${runs.length === 1 ? "run" : "runs"} · ${failed} failed`;
};
