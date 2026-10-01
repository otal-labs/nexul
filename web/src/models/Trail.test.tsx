import { describe, expect, it } from "vitest";

import { mergeLiveStep, mergeLiveSteps, stepLabel, trailSummary, unwrapShellCommand, type ActivityEntry } from "@/models/Trail";

const call: ActivityEntry = { kind: "tool_call", call_id: "c-1", tool: "Bash", summary: "go test", at: "2026-09-18T10:00:00Z" };
const result: ActivityEntry = { ...call, kind: "tool_result", detail: '{"input":{},"result":{"content":"ok"}}', at: "2026-09-18T10:00:04Z" };
const text: ActivityEntry = { kind: "text", summary: "Done", at: "2026-09-18T10:00:09Z" };
const read: ActivityEntry = { kind: "tool_call", call_id: "c-2", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-09-18T10:00:01Z" };

describe("stepLabel", () => {
  it("reads `tool: args` for a tool, the command alone for a command, and leaves a text step alone", () => {
    expect(stepLabel(read)).toBe("Read: main.go");
    expect(stepLabel(call)).toBe("go test");
    expect(stepLabel(text)).toBe("Done");
  });

  it("names a read-type file tool by the path or pattern its JSON arguments carry, and keeps today's label otherwise", () => {
    expect(stepLabel({ ...read, summary: '{"file_path":"/home/dev/web/src/models/Trail.tsx"}' })).toBe("Read: /home/dev/web/src/models/Trail.tsx");
    expect(stepLabel({ ...read, tool: "Grep", summary: '{"pattern":"stepLabel","path":"web/src"}' })).toBe("Grep: stepLabel");
    expect(stepLabel({ ...read, tool: "Glob", summary: '{"pattern":"**/*.tsx"}' })).toBe("Glob: **/*.tsx");
    expect(stepLabel({ ...read, summary: '{"file_path":"/home/dev/a-path-cut-sho…' })).toBe('Read: {"file_path":"/home/dev/a-path-cut-sho…');
  });

  it("names an MCP call by its server and tool, whichever way the harness spells it, without the arguments", () => {
    expect(stepLabel({ ...read, tool: "mcp__nexul__skill_get", summary: '{"name":"nexul-memory"}' })).toBe("Nexul · skill_get");
    expect(stepLabel({ ...read, tool: "nexul · computer_setup_update", summary: "nexul · computer_setup_update" })).toBe("Nexul · computer_setup_update");
  });

  it("falls back to the tool name alone when the call carries no server and no arguments of its own", () => {
    expect(stepLabel({ ...read, tool: "skill_get", summary: "skill_get" })).toBe("skill_get");
  });

  it("drops the shell wrapper from a command and the failed marker from any step", () => {
    expect(stepLabel({ ...call, tool: "Shell", summary: `/bin/bash -lc "rg -n 'Nexul MCP' /home/dev"` })).toBe("rg -n 'Nexul MCP' /home/dev");
    expect(stepLabel({ ...result, summary: "go test ./... · failed" })).toBe("go test ./...");
  });
});

describe("unwrapShellCommand", () => {
  it.each([
    ["a bash login wrapper in double quotes", `/bin/bash -lc "cat /etc/hosts"`, "cat /etc/hosts"],
    ["a bare bash wrapper in single quotes", `bash -lc 'go test ./...'`, "go test ./..."],
    ["an sh wrapper with no quotes", "sh -c ls", "ls"],
    ["nested quotes inside the wrapper", String.raw`/bin/bash -lc "rg -n 'a|b' \"$HOME\""`, String.raw`rg -n 'a|b' \"$HOME\"`],
    ["a wrapper cut short before its closing quote", `/bin/bash -lc "rg -n 'Nexul MCP|skill_get' /home/onik/.codex/mem…`, "rg -n 'Nexul MCP|skill_get' /home/onik/.codex/mem…"],
    ["no wrapper", "go test ./...", "go test ./..."],
    ["a command that only mentions bash", "bash scripts/build.sh", "bash scripts/build.sh"],
    ["empty", "", ""],
  ])("%s", (_name, command, want) => {
    expect(unwrapShellCommand(command)).toBe(want);
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
