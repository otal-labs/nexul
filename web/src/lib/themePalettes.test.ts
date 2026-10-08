import { beforeEach, describe, expect, it } from "vitest";

import { ThemeId } from "@/enums/Theme";
import { applyThemePalette, THEME_DEFINITIONS } from "@/lib/themePalettes";

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
    expect(root.style.getPropertyValue("--shape-md")).toBe("0px");
    expect(root.style.getPropertyValue("--foreground")).toBe("oklch(1.0 0.0 0.0)");
  });

  it("switching to Ember removes Brutalism's --shape-md and --foreground", () => {
    applyThemePalette(ThemeId.Brutalism, "dark");
    applyThemePalette(ThemeId.Ember, "dark");
    const root = document.documentElement;
    expect(root.style.getPropertyValue("--shape-md")).toBe("");
    expect(root.style.getPropertyValue("--foreground")).toBe("");
  });

  it("Console leaves no overridden vars on the root", () => {
    applyThemePalette(ThemeId.Console, "light");
    const root = document.documentElement;
    expect(root.getAttribute("style")).toBeFalsy();
  });
});
