// @vitest-environment node
import { describe, expect, it } from "vitest";

import { fitTail, fitWords } from "@/utils/FitTextUtility";

const chars = (text: string) => text.length;

describe("fitTail", () => {
  it.each([
    [32, "build-farm-eu-central-07-node-14"],
    [30, "build-farm-eu-central-…node-14"],
    [16, "build-fa…node-14"],
    [3, "…node-14"],
  ])("at %i cells", (width, want) => {
    expect(fitTail("build-farm-eu-central-07-", "node-14", width, chars)).toBe(want);
  });
});

describe("fitWords", () => {
  it.each([
    [40, "Customer portal preview for the team"],
    [24, "Customer portal preview…"],
    [23, "Customer portal…"],
    [5, "Cust…"],
    [1, "C…"],
  ])("at %i cells", (width, want) => {
    expect(fitWords("Customer portal preview for the team", width, chars)).toBe(want);
  });
});
