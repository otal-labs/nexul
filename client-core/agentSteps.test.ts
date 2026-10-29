// @vitest-environment node
import { describe, expect, it } from "vitest";

import {
  commandText,
  formatElapsed,
  groupHeadline,
  groupSteps,
  liveStepLabel,
  runDirectory,
  segmentSteps,
  stepLabel,
  stepPhrase,
  turnOutcome,
  turnPhrase,
  turnSpanSeconds,
  unwrapShellCommand,
  type AgentStep,
} from "@nexul/client-core/agentSteps";

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

describe("turn clock and outcome", () => {
  it("spans first to last timestamp, skipping missing ones", () => {
    expect(turnSpanSeconds([text("a", ""), tool("Read", "2026-09-18T10:00:00Z"), tool("Edit", "2026-09-18T10:01:43Z"), text("b", "0001-01-01T00:00:00Z")])).toBe(103);
    expect(turnSpanSeconds([text("a")])).toBeNull();
    expect(formatElapsed(103)).toBe("1m 43s");
    expect(formatElapsed(12)).toBe("12s");
    expect(formatElapsed(3725)).toBe("1h 2m");
  });

  it("reads running over everything, then a failed step, then a call that never came back, else done", () => {
    expect(turnOutcome([call, { ...result, summary: "go test · failed" }], true)).toBe("running");
    expect(turnOutcome([call, { ...result, summary: "go test · failed" }], false)).toBe("failed");
    expect(turnOutcome([read, result], false)).toBe("unfinished");
    expect(turnOutcome([result, done], false)).toBe("done");
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

const step = (tool: string, summary: string, kind: AgentStep["kind"] = "tool_result"): AgentStep => ({ kind, tool, summary, at: "" });

describe("stepPhrase", () => {
  it.each([
    ["a file read by its JSON path", step("Read", '{"file_path":"/repo/web/src/BoardOrder.ts"}'), "Read", "BoardOrder.ts"],
    ["a search by its pattern, not its path", step("Grep", '{"pattern":"useSortable","path":"web/src"}'), "Searched for", "useSortable"],
    ["an edit Codex names by a bare path", step("Edit", "web/src/utils/BoardOrder.ts"), "Edited", "BoardOrder.ts"],
    ["a cut-short JSON preview, by what is left of its file name", step("Read", '{"file_path":"/repo/a-pa…'), "Read", "a-pa…"],
    ["a failed command, without the failed marker or its shell wrapper", step("Shell", `/bin/bash -lc "go test ./..." · failed`), "Ran", "go test ./..."],
    ["an MCP call by server and tool", step("mcp__github__create_pull_request", '{"title":"x"}'), "Called", "Github · create_pull_request"],
    ["any other tool by its name", step("WebFetch", '{"url":"https://example.com"}'), "Used", "WebFetch"],
  ])("%s", (_name, s, verb, object) => {
    expect(stepPhrase(s, false)).toEqual({ verb, object });
  });

  it("speaks in the present while the step runs", () => {
    expect(stepPhrase(step("Bash", "bun run lint", "tool_call"), true)).toEqual({ verb: "Running", object: "bun run lint" });
  });
});

describe("liveStepLabel", () => {
  it("names the newest unanswered call, and reads as thinking between steps or before the first", () => {
    expect(liveStepLabel([step("Read", "a.go"), step("Bash", "go vet", "tool_call")])).toBe("Running go vet");
    expect(liveStepLabel([step("Bash", "go vet")])).toBe("Thinking");
    expect(liveStepLabel([])).toBe("Thinking");
  });
});

describe("groupSteps and turnPhrase", () => {
  const turn = [
    step("Read", '{"file_path":"a/BoardColumn.tsx"}'),
    step("Grep", '{"pattern":"useSortable"}'),
    step("Read", '{"file_path":"a/BoardColumn.tsx"}'),
    step("Bash", "bun run test"),
    text("Swapping the comparator."),
    step("Edit", "a/BoardOrder.ts"),
    step("Edit", "a/BoardOrder.test.ts"),
    step("mcp__github__create_pull_request", "{}"),
  ];

  it("folds neighbouring reads and searches, neighbouring edits and neighbouring commands, leaving every other step alone", () => {
    expect(groupSteps([...turn, step("Bash", "make lint"), step("Shell", "git status"), step("mcp__nexul__account_get", "{}")]).map((g) => [g.act, g.entries.length])).toEqual([
      ["explore", 3],
      ["command", 1],
      ["say", 1],
      ["edit", 2],
      ["tool", 1],
      ["command", 2],
      ["tool", 1],
    ]);
  });

  it("heads a group with what it did, in the present while one of its steps runs", () => {
    expect(groupHeadline("explore", turn.slice(0, 3), false)).toBe("Explored 1 file, 1 search");
    expect(groupHeadline("edit", turn.slice(5, 7), true)).toBe("Editing 2 files");
    expect(groupHeadline("command", [step("Bash", "a"), step("Bash", "b")], false)).toBe("Ran 2 commands");
  });

  it("counts a file read twice once, leaves the Agent's sentences out, and says nothing for a turn of sentences", () => {
    expect(turnPhrase(turn)).toBe("explored 1 file, 1 search, edited 2 files, ran 1 command, used 1 tool");
    expect(turnPhrase([text("Looking.")])).toBe("");
  });
});

describe("runDirectory and commandText", () => {
  const root = "/home/dev/.t3/worktrees/nexul/fix-the-board";
  const cmds = [
    step("Shell", `/bin/bash -lc "cd ${root} && git status"`),
    step("Bash", `cd ${root}; bun run lint`),
    step("Bash", `cd ${root}/web && bun run test · failed`),
    step("Bash", "cd web && bun run typecheck"),
    step("Read", `{"file_path":"${root}/a.go"}`),
  ];

  it("picks the absolute directory most commands start in or under, and none when no command starts with one", () => {
    expect(runDirectory(cmds)).toBe(root);
    expect(runDirectory([step("Bash", "cd web && ls"), step("Bash", "go test")])).toBeNull();
    expect(runDirectory([step("Bash", `cd "/tmp/a b" && ls`)])).toBe("/tmp/a b");
  });

  it.each([
    ["a cd into the run's directory, after && or ;", 0, "git status"],
    ["a cd joined by a semicolon", 1, "bun run lint"],
    ["a cd below it, made relative", 2, "cd web && bun run test"],
    ["a relative cd, left whole", 3, "cd web && bun run typecheck"],
  ])("%s", (_name, i, want) => {
    expect(commandText(cmds[i]!, root)).toBe(want);
  });

  it("leaves a cd elsewhere, a bare cd, and every command when there is no run directory", () => {
    expect(commandText(step("Bash", "cd /etc && cat hosts"), root)).toBe("cd /etc && cat hosts");
    expect(commandText(step("Bash", `cd ${root}`), root)).toBe(`cd ${root}`);
    expect(commandText(cmds[0]!, null)).toBe(`cd ${root} && git status`);
  });

  it("cuts the run's directory in the phrase, the live label and a lone step's label", () => {
    expect(stepPhrase(cmds[1]!, false, root)).toEqual({ verb: "Ran", object: "bun run lint" });
    expect(liveStepLabel([...cmds.slice(0, 2), step("Bash", `cd ${root} && go vet ./...`, "tool_call")])).toBe("Running go vet ./...");
    expect(stepLabel(cmds[0]!)).toBe("git status");
  });
});
