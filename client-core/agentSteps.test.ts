import { describe, expect, it } from "vitest";

import { formatElapsed, segmentSteps, stepLabel, turnCounts, turnSpanSeconds, turnSummary, unwrapShellCommand, type AgentStep } from "@nexul/client-core/agentSteps";

const call: AgentStep = { kind: "tool_call", call_id: "c-1", tool: "Bash", summary: "go test", at: "2026-09-18T10:00:00Z" };
const result: AgentStep = { ...call, kind: "tool_result", detail: '{"input":{},"result":{"content":"ok"}}', at: "2026-09-18T10:00:04Z" };
const done: AgentStep = { kind: "text", summary: "Done", at: "2026-09-18T10:00:09Z" };
const read: AgentStep = { kind: "tool_call", call_id: "c-2", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-09-18T10:00:01Z" };
const text = (s: string, at = ""): AgentStep => ({ kind: "text", summary: s, detail: s, at });
const tool = (tool: string, at = ""): AgentStep => ({ kind: "tool_result", call_id: at || tool, tool, summary: "x", at });

describe("stepLabel", () => {
  it("reads `tool: args` for a tool, the command alone for a command, and leaves a text step alone", () => {
    expect(stepLabel(read)).toBe("Read: main.go");
    expect(stepLabel(call)).toBe("go test");
    expect(stepLabel(done)).toBe("Done");
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
    ["a wrapper cut short before its closing quote", `/bin/bash -lc "rg -n 'Nexul MCP|skill_get' /home/dev/.codex/mem…`, "rg -n 'Nexul MCP|skill_get' /home/dev/.codex/mem…"],
    ["no wrapper", "go test ./...", "go test ./..."],
    ["a command that only mentions bash", "bash scripts/build.sh", "bash scripts/build.sh"],
    ["empty", "", ""],
  ])("%s", (_name, command, want) => {
    expect(unwrapShellCommand(command)).toBe(want);
  });
});

describe("turn counts and clock", () => {
  it("counts tools, commands, and the Agent's sentences", () => {
    const counts = turnCounts([text("a"), tool("Read"), tool("Bash"), tool("Edit"), { kind: "tool_call", tool: "Bash", summary: "ls", at: "" }]);
    expect(counts).toEqual({ tools: 2, commands: 2, updates: 1 });
    expect(turnSummary(counts)).toBe("Used 2 tools, ran 2 commands, and received 1 update");
    expect(turnSummary({ tools: 1, commands: 0, updates: 0 })).toBe("Used 1 tool");
    expect(turnSummary({ tools: 0, commands: 3, updates: 1 })).toBe("Ran 3 commands and received 1 update");
    expect(turnSummary({ tools: 0, commands: 0, updates: 0 })).toBe("No steps yet");
  });

  it("spans first to last timestamp, skipping missing ones", () => {
    expect(turnSpanSeconds([text("a", ""), tool("Read", "2026-09-18T10:00:00Z"), tool("Edit", "2026-09-18T10:01:43Z"), text("b", "0001-01-01T00:00:00Z")])).toBe(103);
    expect(turnSpanSeconds([text("a")])).toBeNull();
    expect(formatElapsed(103)).toBe("1m 43s");
    expect(formatElapsed(12)).toBe("12s");
    expect(formatElapsed(3725)).toBe("1h 2m");
  });
});

describe("segmentSteps", () => {
  it("folds the tools between two messages into a turn that ends at the message, and reads each message as prose", () => {
    expect(segmentSteps([read, done, result], false)).toEqual([
      { kind: "turn", entries: [read], running: false, from: null, until: done.at },
      { kind: "reply", entry: done },
      { kind: "turn", entries: [result], running: false, from: done.at, until: null },
    ]);
  });

  it("keeps a running turn open after the last message, and has nothing before the first step", () => {
    expect(segmentSteps([read, done], true).at(-1)).toEqual({ kind: "turn", entries: [], running: true, from: done.at, until: null });
    expect(segmentSteps([], true)).toEqual([]);
  });
});
