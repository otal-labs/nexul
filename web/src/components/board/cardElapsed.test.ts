import { describe, expect, it } from "vitest";

import { cardElapsed } from "@/components/board/cardElapsed";

describe("cardElapsed", () => {
  it.each([
    [45, "45s"],
    [60, "1m"],
    [986, "16m"],
    [3599, "59m"],
    [3600, "1h"],
    [7380, "2h"],
  ])("shows %i seconds as %s", (seconds, shown) => {
    expect(cardElapsed(seconds)).toBe(shown);
  });
});
