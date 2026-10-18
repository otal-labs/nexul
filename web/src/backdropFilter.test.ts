// @vitest-environment node
import { describe, expect, it } from "vitest";

import css from "@/index.css?raw";

describe("index.css backdrop-filter", () => {
  it("writes only the unprefixed property", () => {
    // Lightning CSS treats the prefixed twin as the same property, so a later -webkit- line erases the unprefixed one in the build; the build adds the prefix itself.
    expect(css).toMatch(/backdrop-filter:/);
    expect(css).not.toMatch(/-webkit-backdrop-filter/);
  });
});
