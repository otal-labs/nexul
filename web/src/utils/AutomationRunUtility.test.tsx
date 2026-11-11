import { describe, expect, it } from "vitest";

import { AutomationRunOutcome } from "@/enums/Automation";
import type { AutomationRun } from "@/models/AutomationRun";
import { RUNS_PAGE, automationLogLine, automationRunDays, automationRunsLine } from "@/utils/AutomationRunUtility";

const run = (id: string, started_at: string, outcome: AutomationRunOutcome = AutomationRunOutcome.Success): AutomationRun => ({
  id,
  automation_id: "a1",
  event_topic: "ticket.created",
  event_id: id,
  outcome,
  started_at,
  finished_at: started_at,
  duration_ms: 1,
  logs: "",
  created_at: started_at,
});

describe("automationRunDays", () => {
  it("splits newest-first runs into one group per local day", () => {
    const now = new Date(2026, 9, 30, 15);
    const days = automationRunDays(
      [run("a", new Date(2026, 9, 30, 9).toISOString()), run("b", new Date(2026, 9, 29, 23).toISOString()), run("c", new Date(2026, 9, 29, 1).toISOString())],
      now,
    );
    expect(days.map((day) => [day.label, day.runs.map((r) => r.id)])).toEqual([
      ["Today", ["a"]],
      ["Yesterday", ["b", "c"]],
    ]);
  });
});

describe("automationRunsLine", () => {
  it("names a full page as the newest runs, never as a total", () => {
    const full = Array.from({ length: RUNS_PAGE }, (_, i) => run(`r${i}`, "2026-10-30T10:00:00Z", i < 7 ? AutomationRunOutcome.Failure : AutomationRunOutcome.Success));
    expect(automationRunsLine(full, false)).toBe("Newest 50 runs · 7 failed");
    expect(automationRunsLine(full, true)).toBe("Newest 50 failures");
  });

  it("counts a short list as it is", () => {
    expect(automationRunsLine([run("a", "2026-10-30T10:00:00Z", AutomationRunOutcome.Crash)], true)).toBe("1 failure");
  });
});

describe("automationLogLine", () => {
  it.each([
    { raw: "plain text", want: { message: "plain text", fields: "" } },
    { raw: "[1,2]", want: { message: "[1,2]", fields: "" } },
    { raw: '{"level":"info"}', want: { message: '{"level":"info"}', fields: "" } },
    { raw: '{"level":"warn","message":"slow","ms":812,"tags":["a"]}', want: { level: "warn", message: "slow", fields: 'ms=812 tags=["a"]' } },
  ])("$raw", ({ raw, want }) => {
    expect(automationLogLine(raw)).toEqual({ time: undefined, level: undefined, ...want });
  });

  it("reads the SDK's ts as a clock time", () => {
    expect(automationLogLine('{"ts":"2026-11-11T16:29:04.112Z","message":"ok"}').time).toMatch(/^\d{2}:\d{2}:\d{2}$/);
  });
});
