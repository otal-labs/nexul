import { describe, expect, it } from "vitest";

import { onlyComputersFull, waitingWhy, type WaitingRun } from "@nexul/client-core/playQueue";

const waiting = (...why: string[]): WaitingRun => ({
  status: "queued",
  reason: "no_computer_ready",
  target_type: "ticket",
  waiting_on: why.map((w) => ({ why: w, text: w })),
});

describe("onlyComputersFull", () => {
  it("counts a computer the auto play's rule left out as no obstacle to waiting for a busy one", () => {
    expect(onlyComputersFull(waiting("full", "rule_out"))).toBe(true);
    expect(waitingWhy(waiting("full", "rule_out"))).toBe("computers busy");
    expect(onlyComputersFull(waiting("full", "no_auto_plays"))).toBe(true);
  });

  it("is false with nothing busy, or with a computer that is down", () => {
    expect(onlyComputersFull(waiting("rule_out"))).toBe(false);
    expect(onlyComputersFull(waiting("full", "offline"))).toBe(false);
  });
});
