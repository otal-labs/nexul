import { describe, expect, it } from "vitest";

import { AutomationRunOutcome } from "@/enums/Automation";
import type { AutomationRun } from "@/models/AutomationRun";
import { RUNS_PAGE, automationLogLine, automationRunDays, automationRunSummary, automationRunsLine } from "@/utils/AutomationRunUtility";

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
    expect(automationRunsLine(full, false)).toBe("Newest 50 runs · 7 failed · median 1ms");
    expect(automationRunsLine(full, true)).toBe("Newest 50 failures");
  });

  it("counts a short list as it is", () => {
    expect(automationRunsLine([run("a", "2026-10-30T10:00:00Z", AutomationRunOutcome.Crash)], true)).toBe("1 failure");
  });

  it("says none failed and takes the lower middle duration of an even list", () => {
    const runs = [5, 1, 40, 3].map((ms, i) => ({ ...run(`r${i}`, "2026-10-30T10:00:00Z"), duration_ms: ms }));
    expect(automationRunsLine(runs, false)).toBe("4 runs · 0 failed · median 3ms");
  });
});

describe("automationRunSummary", () => {
  it.each([
    { logs: "", want: null },
    { logs: "plain text\n", want: { message: "plain text", fields: "" } },
    {
      logs: '{"message":"first"}\n{"ts":"2026-11-24T18:23:21Z","run_id":"01a2","message":"queued the review","pr":6,"ticketId":"01a1","state":"queued","reason":"","tags":["a"]}\n',
      want: { message: "queued the review", fields: "pr=6 state=queued" },
    },
  ])("$logs", ({ logs, want }) => {
    expect(automationRunSummary(logs)).toEqual(want);
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
