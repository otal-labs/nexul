import { describe, expect, it } from "vitest";

import { entranceDelayMs } from "@/components/runner/motion";

describe("entranceDelayMs", () => {
  it("staggers the first eight rows at 24ms increments", () => {
    expect(entranceDelayMs(0)).toBe(0);
    expect(entranceDelayMs(1)).toBe(24);
    expect(entranceDelayMs(7)).toBe(168);
  });

  it("stops staggering beyond the eighth row so a long fleet mounts together", () => {
    expect(entranceDelayMs(8)).toBe(0);
    expect(entranceDelayMs(50)).toBe(0);
  });
});
