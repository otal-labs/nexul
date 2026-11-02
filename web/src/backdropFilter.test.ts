// @vitest-environment node
import { describe, expect, it } from "vitest";

import css from "@/index.css?raw";

// The top-level rule each backdrop-filter sits in, by the last rule opened at column 0 before it.
const blurredRules = () =>
  [...css.matchAll(/^\s+backdrop-filter:/gm)].map((match) => {
    const before = css.slice(0, match.index);
    return [...before.matchAll(/^(\S[^\n{]*?)\s*\{/gm)].at(-1)?.[1];
  });

describe("index.css backdrop-filter", () => {
  it("writes only the unprefixed property", () => {
    // Lightning CSS treats the prefixed twin as the same property, so a later -webkit- line erases the unprefixed one in the build; the build adds the prefix itself.
    expect(css).toMatch(/backdrop-filter:/);
    expect(css).not.toMatch(/-webkit-backdrop-filter/);
  });

  it("blurs only the scrim and the one frosted card, never the everyday panels or dialogs", () => {
    expect(blurredRules()).toEqual([".glass-overlay", "@utility panel-frosted"]);
  });
});
