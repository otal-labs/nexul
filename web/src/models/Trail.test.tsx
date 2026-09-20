import { describe, expect, it } from "vitest";

import { mergeLiveStep, mergeLiveSteps, stepLabel, trailSummary, type ActivityEntry } from "@/models/Trail";

const call: ActivityEntry = { kind: "tool_call", call_id: "c-1", tool: "Bash", summary: "go test", at: "2026-09-18T10:00:00Z" };
const result: ActivityEntry = { ...call, kind: "tool_result", detail: '{"input":{},"result":{"content":"ok"}}', at: "2026-09-18T10:00:04Z" };
const text: ActivityEntry = { kind: "text", summary: "Done", at: "2026-09-18T10:00:09Z" };
const read: ActivityEntry = { kind: "tool_call", call_id: "c-2", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-09-18T10:00:01Z" };

describe("stepLabel", () => {
  it("reads `tool: args` for a tool, the command alone for a command, and leaves a text step alone", () => {
    expect(stepLabel(read)).toBe('Read: {"file_path":"main.go"}');
    expect(stepLabel(call)).toBe("go test");
    expect(stepLabel(text)).toBe("Done");
  });
});

describe("mergeLiveStep", () => {
  it("leaves the list alone without a live step", () => {
    expect(mergeLiveStep([call], null)).toEqual([call]);
  });

  it("replaces the step sharing the live step's call id", () => {
    expect(mergeLiveStep([call, text], result)).toEqual([result, text]);
  });

  it("appends a step with a new call id or none, once", () => {
    expect(mergeLiveStep([call], text)).toEqual([call, text]);
    expect(mergeLiveStep([call, text], text)).toEqual([call, text]);
  });
});

describe("mergeLiveSteps", () => {
  it("folds the live steps onto the list once each, and never steps a result back to its call", () => {
    expect(mergeLiveSteps([call], [result, text])).toEqual([result, text]);
    expect(mergeLiveSteps([result, text], [call, result, text])).toEqual([result, text]);
    expect(mergeLiveSteps([], [call, read])).toEqual([call, read]);
  });
});

describe("trailSummary", () => {
  it("names the latest step while running, the reason once ended badly, the state otherwise", () => {
    expect(trailSummary("running", "", call)).toBe("go test");
    expect(trailSummary("running", "", null)).toBe("Running…");
    expect(trailSummary("failed", "Harness offline", call)).toBe("Failed · Harness offline");
    expect(trailSummary("interrupted", "", undefined)).toBe("Interrupted");
    expect(trailSummary("done", "", call)).toBe("Done");
  });
});

describe("trailSummary error truncation", () => {
  it("keeps a failed row to the first line and eighty characters", () => {
    const long = `${"x".repeat(100)}\nsecond line`;
    expect(trailSummary("failed", long, null)).toBe(`Failed · ${"x".repeat(80)}…`);
  });
});
