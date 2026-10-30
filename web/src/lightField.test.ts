import { describe, expect, it } from "vitest";

import css from "@/index.css?raw";
import still from "@/components/showcase/ShowcaseField.tsx?raw";

// A literal colour carries its own hue; a relative `oklch(from var(--brand) …)` keeps the palette's.
const LITERAL_COLOUR = /#[0-9a-f]{3,8}\b|\b(?:rgb|hsl|hwb|lab|lch)a?\(|\boklch\((?!from )/gi;
const literals = (text: string): string[] => text.match(LITERAL_COLOUR) ?? [];

const fieldTokens = [...css.matchAll(/--field-[\w-]+:\s*([^;]+);/g)].map((m) => m[1]!);
const declaredIn = (block: string): string[] => [...block.matchAll(/(--field-[\w-]+):/g)].map((m) => m[1]!).sort();
const lightMode = css.match(/\n:root \{([\s\S]*?)\n\}/)?.[1] ?? "";
const darkMode = css.match(/\n\.dark \{([\s\S]*?)\n\}/)?.[1] ?? "";
const lightField = css.match(/\.light-field \{([\s\S]*?)\n\}/)?.[1] ?? "";

describe("the light field's colours", () => {
  it("are declared in both modes", () => {
    expect(declaredIn(lightMode)).not.toEqual([]);
    expect(declaredIn(darkMode)).toEqual(declaredIn(lightMode));
  });

  it("are never a literal colour, so every palette retints them", () => {
    expect(fieldTokens.filter((value) => literals(value).length > 0)).toEqual([]);
  });

  // The mid glow lit the middle, under the panels, where their flat fill turned its falloff into a disc.
  it("are painted from the warm and low field tokens in the app shell's gradients, never the mid glow", () => {
    const stops = [...lightField.matchAll(/radial-gradient\([^,]+,\s*([^,]+),/g)].map((m) => m[1]);
    expect(stops).toEqual(["var(--field-warm)", "var(--field-low)"]);
  });

  it("are painted from the field tokens on the showcase surfaces", () => {
    expect(literals(still)).toEqual([]);
    expect(still.match(/oklch\(from var\(--field-\w+\)/g)).toHaveLength(3);
  });
});
