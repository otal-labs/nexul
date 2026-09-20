import { describe, expect, it } from "vitest";

import type { ActivityEntry, Trail } from "@/models/Trail";
import { formatElapsed, pullRequestLink, segmentTranscript, turnCounts, turnSpanSeconds, turnSummary } from "@/utils/TrailTranscriptUtility";

const trail = { play_label: "Fix with AI", custom_instructions: "", started_at: "2026-09-18T10:00:00Z" } as Trail;
const text = (s: string, at = ""): ActivityEntry => ({ kind: "text", summary: s, detail: s, at });
const tool = (tool: string, at = ""): ActivityEntry => ({ kind: "tool_result", call_id: at || tool, tool, summary: "x", at });

describe("segmentTranscript", () => {
  it("a finished run: bubble, one turn, the closing text as the reply", () => {
    const segments = segmentTranscript(trail, [text("a"), tool("Read"), text("b")], "done", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "reply"]);
    expect(segments[0]).toMatchObject({ body: "Started Fix with AI", at: "2026-09-18T10:00:00Z" });
    expect(segments[1]).toMatchObject({ entries: [text("a"), tool("Read")], running: false, until: "" });
  });

  it("a turn that ends in a reply spans up to the reply, not its last row", () => {
    const segments = segmentTranscript(trail, [tool("Read", "2026-09-18T10:00:00Z"), text("done", "2026-09-18T10:00:09Z")], "done", null);
    const turn = segments[1] as { entries: ActivityEntry[]; until: string | null };
    expect(turn.until).toBe("2026-09-18T10:00:09Z");
    expect(turnSpanSeconds(turn.entries, turn.until)).toBe(9);
  });

  it("while running the closing text stays in the turn and the turn runs", () => {
    const segments = segmentTranscript(trail, [text("a"), tool("Read"), text("b")], "running", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn"]);
    expect(segments[1]).toMatchObject({ running: true });
    expect((segments[1] as { entries: ActivityEntry[] }).entries).toHaveLength(3);
  });

  it("a fresh running run gets an empty running turn so the clock shows", () => {
    expect(segmentTranscript(trail, [], "starting", null)).toEqual([
      { kind: "user", body: "Started Fix with AI", at: "2026-09-18T10:00:00Z" },
      { kind: "turn", entries: [], running: true, from: "2026-09-18T10:00:00Z", until: null },
    ]);
  });

  it("the latest question splits the turns, with the answer as the starter's bubble", () => {
    const q = { request_id: "r", questions: [{ id: "q1", text: "Go?", options: [] }], asked_at: "", answer: { answers: { q1: { text: "yes please" } } } };
    const asked: ActivityEntry = { kind: "question", call_id: "c", tool: "AskUserQuestion", summary: "Go?", at: "" };
    const segments = segmentTranscript(trail, [tool("Read"), asked, tool("Edit"), text("done")], "done", q);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "question", "user", "turn", "reply"]);
    expect(segments[3]).toMatchObject({ body: "Answered: yes please", at: null });
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
    const segments = segmentTranscript(trail, [tool("Read")], "waiting", q);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "question"]);
    expect(segments[1]).toMatchObject({ running: false });
  });

  it("a note closes the turn, hands its closing text over as the reply, and stands on its own", () => {
    const segments = segmentTranscript(trail, [tool("Read"), text("bye"), { kind: "note", summary: "Run stopped by onik.", at: "" }], "interrupted", null);
    expect(segments.map((s) => s.kind)).toEqual(["user", "turn", "reply", "note"]);
    expect(segments[2]).toMatchObject({ entry: text("bye") });
    expect(segments[3]).toMatchObject({ text: "Run stopped by onik." });
  });

  it("the first turn's clock runs from the run's start to its end", () => {
    const ended = { ...trail, ended_at: "2026-09-18T10:00:26Z" } as Trail;
    const segments = segmentTranscript(ended, [tool("Read", "2026-09-18T10:00:22Z"), text("done", "2026-09-18T10:00:26Z")], "done", null);
    const turn = segments[1] as { entries: ActivityEntry[]; from: string | null; until: string | null };
    expect(turn.from).toBe("2026-09-18T10:00:00Z");
    expect(turn.until).toBe("2026-09-18T10:00:26Z");
    expect(turnSpanSeconds(turn.entries, turn.until, turn.from)).toBe(26);
  });

  it("custom instructions ride under the started line", () => {
    const segments = segmentTranscript({ ...trail, custom_instructions: "Docs only." }, [], "done", null);
    expect(segments[0]).toMatchObject({ body: "Started Fix with AI\n\nDocs only." });
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

  it("finds the pull request the reply links", () => {
    expect(pullRequestLink("Opened https://github.com/otal-labs/nexul/pull/51 for review.")).toEqual({
      url: "https://github.com/otal-labs/nexul/pull/51",
      label: "otal-labs/nexul#51",
    });
    expect(pullRequestLink("No PR this time.")).toBeNull();
  });
});
