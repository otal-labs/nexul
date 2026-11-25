import { describe, expect, it } from "vitest";

import { appendLogLines, lastLogSeq } from "@nexul/client-core/deployLog";

const line = (seq: number, ts: number) => ({ seq, ts, text: `line ${seq}` });

describe("appendLogLines", () => {
  it("merges every batch into the order one whole fetch returns, a late batch included", () => {
    const whole = [line(1, 20), line(2, 20), line(3, 30), line(4, 10), line(5, 30)].sort((a, b) => a.ts - b.ts || a.seq - b.seq);
    let cached = appendLogLines([], [line(1, 20), line(2, 20)]);
    cached = appendLogLines(cached, [line(3, 30)]);
    cached = appendLogLines(cached, [line(4, 10), line(5, 30)]);
    expect(cached).toEqual(whole);
    expect(lastLogSeq(cached)).toBe(5);
  });

  it("keeps a line two overlapping fetches both returned once, and hands back the same log when nothing is new", () => {
    const cached = appendLogLines([line(1, 1)], [line(2, 2)]);
    expect(appendLogLines(cached, [line(2, 2), line(3, 3)]).map((l) => l.seq)).toEqual([1, 2, 3]);
    expect(appendLogLines(cached, [line(2, 2)])).toBe(cached);
  });
});
