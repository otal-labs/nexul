import { describe, expect, it } from "vitest";

import { lastRunDate } from "@/components/agent/lastRunDate";

const now = new Date(2026, 10, 27, 15, 0);

describe("lastRunDate", () => {
  it.each([
    [new Date(2026, 10, 27, 9, 0), "Today"],
    [new Date(2026, 10, 26, 23, 0), "Yesterday"],
    [new Date(2026, 10, 20, 9, 0), "20 Nov"],
    [new Date(2025, 9, 28, 9, 0), "28 Oct 2025"],
  ])("reads %s as %s, never as a relative figure beside dates", (date, expected) => {
    expect(lastRunDate(date.toISOString(), now)).toBe(expected);
  });
});
