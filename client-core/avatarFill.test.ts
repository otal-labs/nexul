import { describe, expect, it } from "vitest";

import { avatarFill, avatarOklch } from "@nexul/client-core/avatarFill";

// WCAG relative luminance of an oklch colour, clipped to sRGB as a browser does.
const luminance = (l: number, c: number, hue: number): number => {
  const [a, b] = [c * Math.cos((hue * Math.PI) / 180), c * Math.sin((hue * Math.PI) / 180)];
  const [lc, mc, sc] = [(l + 0.3963377774 * a + 0.2158037573 * b) ** 3, (l - 0.1055613458 * a - 0.0638541728 * b) ** 3, (l - 0.0894841775 * a - 1.291485548 * b) ** 3];
  const linear = [4.0767416621 * lc - 3.3077115913 * mc + 0.2309699292 * sc, -1.2684380046 * lc + 2.6097574011 * mc - 0.3413193965 * sc, -0.0041960863 * lc - 0.7034186147 * mc + 1.707614701 * sc];
  const [r, g, bl] = linear.map((v) => Math.min(1, Math.max(0, v)));
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * bl!;
};

const seeds = Array.from({ length: 50 }, (_, i) => `person_${i}_${String.fromCharCode(97 + (i % 26))}${i * 7919}`);
const huesOf = (list: string[]): number[] => list.map((seed) => avatarOklch(seed)[2]);
const gap = (a: number, b: number): number => Math.min(Math.abs(a - b), 360 - Math.abs(a - b));

describe("avatarFill", () => {
  it("keeps white initials at 4.5:1 or better on every seeded fill", () => {
    const failing = seeds.filter((seed) => 1.05 / (luminance(...avatarOklch(seed)) + 0.05) < 4.5);
    expect(failing).toEqual([]);
  });

  it("is one flat colour with no second stop", () => {
    for (const seed of seeds) expect(avatarFill(seed)).toMatch(/^oklch\([\d.]+ [\d.]+ [\d.]+\)$/);
  });

  it("covers the wheel, so people read as different colours rather than one", () => {
    const buckets = new Set(huesOf(seeds).map((hue) => Math.floor(hue / 36)));
    expect(buckets.size).toBeGreaterThanOrEqual(8);
  });

  it("gives the same person the same fill", () => {
    expect(avatarFill("bob_nw")).toBe(avatarFill("bob_nw"));
  });

  it("keeps the demo workspace's people on nine distinct hues, none under 30 degrees apart", () => {
    const hues = huesOf(["aleksandra_nw", "alice_nw", "bartholomew_nw", "bob_nw", "jo_nw", "lena_nw", "noor_nw", "priya_nw", "sam_nw"]);
    const closest = Math.min(...hues.flatMap((hue, i) => hues.slice(i + 1).map((other) => gap(hue, other))));
    expect(closest).toBeGreaterThanOrEqual(30);
  });
});
