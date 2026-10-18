// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { StackRequests } from "@/models/Stack";
import { formatShare, minuteAt, missingSince, requestTotals } from "@/utils/RequestsUtility";

const hour = (requests: (number | null)[], errors: (number | null)[] = requests.map((r) => (r === null ? null : 0))): StackRequests => ({
  counted: true,
  from: "2026-10-16T09:43:00.000Z",
  requests,
  errors_5xx: errors,
});

describe("requestTotals", () => {
  it("sums only the counted minutes and shares the 5xx among them", () => {
    const totals = requestTotals(hour([10, null, 30, 0], [1, null, 3, 0]));
    expect(totals).toEqual({ requests: 40, errors: 4, share: 0.1, counted: 3 });
  });

  it("has no share for an hour with no requests", () => {
    expect(requestTotals(hour([0, 0, null])).share).toBe(0);
  });
});

describe("missingSince", () => {
  const counted = (n: number) => Array.from({ length: n }, () => 1);
  const gap = (n: number) => Array.from({ length: n }, () => null);

  it("names the first minute of a gap that has run five minutes", () => {
    const w = hour([...counted(55), ...gap(5)]);
    expect(missingSince(w)).toBe("2026-10-16T10:38:00.000Z");
    expect(missingSince(w)).toBe(minuteAt(w, 55));
  });

  it("says nothing of a shorter gap, a gap in the middle, or an hour with nothing counted", () => {
    expect(missingSince(hour([...counted(56), ...gap(4)]))).toBeUndefined();
    expect(missingSince(hour([...counted(20), ...gap(10), ...counted(30)]))).toBeUndefined();
    expect(missingSince(hour(gap(60)))).toBeUndefined();
  });
});

describe("formatShare", () => {
  it.each([
    [0, "0%"],
    [0.0004, "<0.1%"],
    [0.004, "0.4%"],
    [0.072, "7.2%"],
    [0.126, "13%"],
    [1, "100%"],
  ])("%s reads %s", (share, text) => {
    expect(formatShare(share)).toBe(text);
  });
});
