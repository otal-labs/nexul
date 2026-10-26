import { avatarFill } from "@/lib/color";

const luminance = (rgb: string): number => {
  const [r, g, b] = (rgb.match(/\d+/g) ?? []).map((v) => {
    const c = Number(v) / 255;
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!;
};

describe("avatarFill", () => {
  it("keeps white initials at 4.5:1 or better on every seeded fill", () => {
    const seeds = Array.from({ length: 50 }, (_, i) => `person_${i}_${String.fromCharCode(97 + (i % 26))}${i * 7919}`);
    const failing = seeds.filter((seed) => 1.05 / (luminance(avatarFill(seed)) + 0.05) < 4.5);
    expect(failing).toEqual([]);
  });

  it("is one flat colour, and the same for the same person", () => {
    expect(avatarFill("bob_nw")).toMatch(/^rgb\(\d+, \d+, \d+\)$/);
    expect(avatarFill("bob_nw")).toBe(avatarFill("bob_nw"));
    expect(avatarFill("bob_a")).not.toBe(avatarFill("bob_b"));
  });
});
