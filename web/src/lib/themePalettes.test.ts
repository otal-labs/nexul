import { beforeEach, describe, expect, it } from "vitest";

import { ThemeId } from "@/enums/Theme";
import { applyThemePalette, getThemeDefinition, THEME_DEFINITIONS } from "@/lib/themePalettes";

const OKLCH_PATTERN = /^oklch\(/;

describe("THEME_DEFINITIONS", () => {
  it("has every ThemeId exactly once", () => {
    const ids = Object.values(ThemeId);
    for (const id of ids) {
      expect(THEME_DEFINITIONS.filter((theme) => theme.id === id)).toHaveLength(1);
    }
    expect(THEME_DEFINITIONS).toHaveLength(ids.length);
  });

  it.each(THEME_DEFINITIONS.filter((theme) => theme.colors !== null))(
    "$label has both light and dark overrides in oklch",
    (theme) => {
      const colors = theme.colors;
      if (!colors) throw new Error("expected colors to be non-null");
      for (const mode of ["light", "dark"] as const) {
        const overrides = colors[mode];
        expect(Object.keys(overrides).length).toBeGreaterThan(0);
        for (const value of Object.values(overrides)) {
          expect(value).toMatch(OKLCH_PATTERN);
        }
      }
    },
  );
});

describe("applyThemePalette", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("style");
    document.documentElement.removeAttribute("data-theme-id");
  });

  it("Brutalism dark writes --shape-md and --foreground", () => {
    applyThemePalette(ThemeId.Brutalism, "dark");
    const root = document.documentElement;
    const brutalism = getThemeDefinition(ThemeId.Brutalism);
    expect(root.style.getPropertyValue("--shape-md")).toBe(brutalism.radius?.md);
    expect(root.style.getPropertyValue("--shape-2xl")).toBe(brutalism.radius?.["2xl"]);
    expect(root.style.getPropertyValue("--foreground")).toBe(brutalism.colors?.dark.foreground);
    expect(root.style.getPropertyValue("--shape-md")).not.toBe("");
    expect(root.style.getPropertyValue("--foreground")).not.toBe("");
  });

  it("switching to Ember removes Brutalism's --shape-md and --foreground", () => {
    applyThemePalette(ThemeId.Brutalism, "dark");
    applyThemePalette(ThemeId.Ember, "dark");
    const root = document.documentElement;
    expect(root.style.getPropertyValue("--shape-md")).toBe("");
    expect(root.style.getPropertyValue("--foreground")).toBe("");
  });

  it("a palette's primary becomes the accent, and the default palette clears it again", () => {
    const root = document.documentElement;
    applyThemePalette(ThemeId.Ocean, "dark");
    expect(root.style.getPropertyValue("--brand")).toBe(getThemeDefinition(ThemeId.Ocean).colors?.dark.primary);
    expect(root.style.getPropertyValue("--brand-foreground")).toBe("var(--primary-foreground)");
    applyThemePalette(ThemeId.Console, "dark");
    expect(root.style.getPropertyValue("--brand")).toBe("");
  });

  it("Console leaves no overridden vars on the root", () => {
    applyThemePalette(ThemeId.Console, "light");
    const root = document.documentElement;
    expect(root.getAttribute("style")).toBeFalsy();
  });
});

// index.css's defaults for the two roles a palette may leave out.
const DEFAULTS = {
  light: { card: "oklch(1 0 0)", destructive: "#c41945" },
  dark: { card: "oklch(0.21 0.005 265)", destructive: "#f26074" },
};

const srgb = (color: string): [number, number, number] => {
  if (color.startsWith("#")) return [1, 3, 5].map((i) => Number.parseInt(color.slice(i, i + 2), 16) / 255) as [number, number, number];
  const [l, c, h] = color.replace(/^oklch\(|\)$/g, "").split(/\s+/).map(Number) as [number, number, number];
  const a = c * Math.cos((h * Math.PI) / 180);
  const b = c * Math.sin((h * Math.PI) / 180);
  const lms = [l + 0.3963377774 * a + 0.2158037573 * b, l - 0.1055613458 * a - 0.0638541728 * b, l - 0.0894841775 * a - 1.291485548 * b].map((x) => x ** 3) as [number, number, number];
  const linear = [
    4.0767416621 * lms[0] - 3.3077115913 * lms[1] + 0.2309699292 * lms[2],
    -1.2684380046 * lms[0] + 2.6097574011 * lms[1] - 0.3413193965 * lms[2],
    -0.0041960863 * lms[0] - 0.7034186147 * lms[1] + 1.707614701 * lms[2],
  ];
  return linear.map((x) => Math.min(1, Math.max(0, x <= 0.0031308 ? 12.92 * x : 1.055 * x ** (1 / 2.4) - 0.055))) as [number, number, number];
};

const luminance = (color: string) =>
  srgb(color)
    .map((x) => (x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4))
    .reduce((sum, x, i) => sum + x * [0.2126, 0.7152, 0.0722][i]!, 0);

const contrast = (a: string, b: string) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x) as [number, number];
  return (hi + 0.05) / (lo + 0.05);
};

describe("destructive in every palette", () => {
  it.each(THEME_DEFINITIONS.flatMap((theme) => (["light", "dark"] as const).map((mode) => ({ label: `${theme.label} ${mode}`, theme, mode }))))(
    "$label keeps error text at 4.5:1 on its cards",
    ({ theme, mode }) => {
      const roles = theme.colors?.[mode];
      const card = roles?.card ?? DEFAULTS[mode].card;
      const destructive = roles?.destructive ?? DEFAULTS[mode].destructive;
      expect(contrast(destructive, card)).toBeGreaterThanOrEqual(4.5);
    },
  );

  it("moves Brutalism's errors off its pure-red accent's hue", () => {
    const brutalism = getThemeDefinition(ThemeId.Brutalism).colors!;
    for (const mode of ["light", "dark"] as const) {
      const hue = (color: string) => Number(color.replace(/^oklch\(|\)$/g, "").split(/\s+/)[2]);
      expect(Math.abs(hue(brutalism[mode].primary!) - hue(brutalism[mode].destructive!))).toBeGreaterThanOrEqual(25);
    }
  });
});
