import { describe, expect, it } from "vitest";

import {
  answersChangedSinceWritten,
  clarificationStatus,
  roundNumber,
  type Clarification,
  type ClarificationQuestion,
  type ClarificationRound,
} from "@/models/Clarification";

const question = (round: number, answered_at: string): ClarificationQuestion => ({
  id: `q-${round}-${answered_at}`, doc_id: "d", round, position: 1, question: "Which?", why: "", options: [],
  multi_select: false, selected: ["A"], text: "", skipped: false, answered_at,
});

const round = (n: number, questions: ClarificationQuestion[], extra: Partial<ClarificationRound> = {}): ClarificationRound => ({
  doc_id: "d", round: n, started_by: "u", trail_id: `t-${n}`, started_at: "2026-10-01T00:00:00Z", running: false,
  anything_else: "", anything_else_reply: "", questions, ...extra,
});

const clarification = (rounds: ClarificationRound[], extra: Partial<Clarification> = {}): Clarification => ({
  rounds, running: false, closed: false, can_close: true, ...extra,
});

// Round 2 found no gaps, so the server's round 3 is the second one that asked something.
const afterNoGaps = clarification([
  round(1, [question(1, "2026-10-01T01:00:00Z")]),
  round(2, [], { no_gaps_at: "2026-10-01T02:00:00Z" }),
  round(3, [question(3, "2026-10-01T03:00:00Z")]),
]);

describe("roundNumber", () => {
  it("counts only the rounds that asked something, in order", () => {
    expect(roundNumber(afterNoGaps, afterNoGaps.rounds[0]!)).toBe(1);
    expect(roundNumber(afterNoGaps, afterNoGaps.rounds[2]!)).toBe(2);
  });
});

describe("clarificationStatus", () => {
  it("names the round by what it asked, and counts only those when closed", () => {
    expect(clarificationStatus(afterNoGaps, true, false).label).toBe("Round 2 answered");
    const closed = { ...afterNoGaps, closed: true, rounds: [afterNoGaps.rounds[0]!, afterNoGaps.rounds[2]!] };
    expect(clarificationStatus(closed, false, false).detail).toBe("2 rounds");
  });

  it("numbers a running round after the ones that asked", () => {
    const running = clarification([...afterNoGaps.rounds, round(4, [], { running: true })], { running: true });
    expect(clarificationStatus(running, true, false).label).toBe("Round 3 running");
  });
});

describe("answersChangedSinceWritten", () => {
  it("ignores answers to questions asked after the doc was written", () => {
    expect(answersChangedSinceWritten(afterNoGaps)).toBe(0);
  });

  it("counts an earlier question answered again after the write", () => {
    const changed = clarification([
      round(1, [question(1, "2026-10-01T03:00:00Z")]),
      round(2, [], { no_gaps_at: "2026-10-01T02:00:00Z" }),
    ]);
    expect(answersChangedSinceWritten(changed)).toBe(1);
  });
});
