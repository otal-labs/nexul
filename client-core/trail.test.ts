import { describe, expect, it } from "vitest";

import { turnSpanSeconds, type AgentStep } from "@nexul/client-core/agentSteps";
import { mergeLiveStep, mergeLiveSteps, segmentTranscript, trailSummary, type TranscriptTrail } from "@nexul/client-core/trail";


const call: AgentStep = { kind: "tool_call", call_id: "c-1", tool: "Bash", summary: "go test", at: "2026-09-18T10:00:00Z" };
const result: AgentStep = { ...call, kind: "tool_result", detail: '{"input":{},"result":{"content":"ok"}}', at: "2026-09-18T10:00:04Z" };
const text: AgentStep = { kind: "text", summary: "Done", at: "2026-09-18T10:00:09Z" };
const read: AgentStep = { kind: "tool_call", call_id: "c-2", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-09-18T10:00:01Z" };

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

  it("does not name a message the starter wrote in T3 as the Agent's latest step", () => {
    const typed: AgentStep = { kind: "user_message", call_id: "m", tool: "T3", summary: "Use two threads", at: "" };
    expect(trailSummary("running", "", typed)).toBe("Running…");
  });
});

describe("trailSummary error truncation", () => {
  it("keeps a failed row to the first line and eighty characters", () => {
    const long = `${"x".repeat(100)}\nsecond line`;
    expect(trailSummary("failed", long, null)).toBe(`Failed · ${"x".repeat(80)}…`);
  });
});

const trail: TranscriptTrail = { play_label: "Fix with AI", custom_instructions: "", started_at: "2026-09-18T10:00:00Z", ended_at: null };
const say = (s: string, at = ""): AgentStep => ({ kind: "text", summary: s, detail: s, at });
const act = (tool: string, at = ""): AgentStep => ({ kind: "tool_result", call_id: at || tool, tool, summary: "x", at });

describe("segmentTranscript", () => {
  it("a finished run: bubble, one turn, the closing text as the reply", () => {
    const segments = segmentTranscript(trail, [say("a"), act("Read"), say("b")], "done", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "reply"]);
    expect(segments[0]).toMatchObject({ body: "Started Fix with AI", at: "2026-09-18T10:00:00Z" });
    expect(segments[1]).toMatchObject({ entries: [say("a"), act("Read")], running: false, until: "" });
  });

  it("a turn that ends in a reply spans up to the reply, not its last row", () => {
    const segments = segmentTranscript(trail, [act("Read", "2026-09-18T10:00:00Z"), say("done", "2026-09-18T10:00:09Z")], "done", null);
    const turn = segments[1] as { entries: AgentStep[]; until: string | null };
    expect(turn.until).toBe("2026-09-18T10:00:09Z");
    expect(turnSpanSeconds(turn.entries, turn.until)).toBe(9);
  });

  it("while running the closing text stays in the turn and the turn runs", () => {
    const segments = segmentTranscript(trail, [say("a"), act("Read"), say("b")], "running", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn"]);
    expect(segments[1]).toMatchObject({ running: true });
    expect((segments[1] as { entries: AgentStep[] }).entries).toHaveLength(3);
  });

  it("a fresh running run gets an empty running turn so the clock shows", () => {
    expect(segmentTranscript(trail, [], "starting", null)).toEqual([
      { kind: "user", body: "Started Fix with AI", at: "2026-09-18T10:00:00Z", via: null },
      { kind: "turn", entries: [], running: true, from: "2026-09-18T10:00:00Z", until: null },
    ]);
  });

  it("the latest question splits the turns, with the answer as the starter's bubble", () => {
    const q = { request_id: "r", questions: [{ id: "q1", text: "Go?", options: [] }], asked_at: "", answer: { answers: { q1: { text: "yes please" } } } };
    const asked: AgentStep = { kind: "question", call_id: "c", tool: "AskUserQuestion", summary: "Go?", at: "" };
    const segments = segmentTranscript(trail, [act("Read"), asked, act("Edit"), say("done")], "done", q);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "question", "user", "turn", "reply"]);
    expect(segments[3]).toMatchObject({ body: "Answered: yes please", at: null });
  });

  it("a message the starter wrote in T3 closes the turn as their own bubble, with its full text and where it was written", () => {
    const typed: AgentStep = { kind: "user_message", call_id: "m", tool: "T3", summary: "Use two", detail: "Use two threads", at: "2026-09-18T10:05:00Z" };
    const segments = segmentTranscript(trail, [act("Read"), say("Which policy?"), typed, act("Edit"), say("done")], "done", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "reply", "user", "turn", "reply"]);
    expect(segments[3]).toEqual({ kind: "user", body: "Use two threads", at: "2026-09-18T10:05:00Z", via: "T3" });
  });

  it("several questions in one request list every answer", () => {
    const q = {
      request_id: "r",
      questions: [{ id: "a", text: "Branch?", options: [] }, { id: "b", text: "Push?", options: [] }],
      asked_at: "",
      answer: { answers: { a: { text: "main" }, b: { selected: ["No"] } } },
    };
    const segments = segmentTranscript(trail, [], "done", q);
    expect(segments[2]).toMatchObject({ body: "Answered:\n- Branch?: main\n- Push?: No" });
  });

  it("an unanswered question while waiting shows the card and nothing runs", () => {
    const q = { request_id: "r", questions: [{ id: "q1", text: "Go?", options: [] }], asked_at: "" };
    const segments = segmentTranscript(trail, [act("Read")], "waiting", q);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "question"]);
    expect(segments[1]).toMatchObject({ running: false });
  });

  it("a note closes the turn, hands its closing text over as the reply, and stands on its own", () => {
    const segments = segmentTranscript(trail, [act("Read"), say("bye"), { kind: "note", summary: "Run stopped by onik.", at: "" }], "interrupted", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "reply", "note"]);
    expect(segments[2]).toMatchObject({ entry: say("bye") });
    expect(segments[3]).toMatchObject({ text: "Run stopped by onik." });
  });

  it("the first turn's clock runs from the run's start to its end", () => {
    const ended = { ...trail, ended_at: "2026-09-18T10:00:26Z" };
    const segments = segmentTranscript(ended, [act("Read", "2026-09-18T10:00:22Z"), say("done", "2026-09-18T10:00:26Z")], "done", null);
    const turn = segments[1] as { entries: AgentStep[]; from: string | null; until: string | null };
    expect(turn.from).toBe("2026-09-18T10:00:00Z");
    expect(turn.until).toBe("2026-09-18T10:00:26Z");
    expect(turnSpanSeconds(turn.entries, turn.until, turn.from)).toBe(26);
  });

  it("custom instructions ride under the started line", () => {
    const segments = segmentTranscript({ ...trail, custom_instructions: "Docs only." }, [], "done", null);
    expect(segments[0]).toMatchObject({ body: "Started Fix with AI\n\nDocs only." });
  });
});
