import { beforeEach, describe, expect, it } from "vitest";

import { ThemeName } from "@/enums/Theme";
import { useThemeStore } from "@/stores/themeStore";

describe("themeStore", () => {
  beforeEach(() => {
    document.documentElement.classList.remove("dark");
    useThemeStore.setState({ theme: ThemeName.Light });
    localStorage.clear();
  });

  it("defaults to light", () => {
    expect(useThemeStore.getState().theme).toBe(ThemeName.Light);
  });

  it("setTheme applies the dark class to documentElement", () => {
    useThemeStore.getState().setTheme(ThemeName.Dark);
    expect(useThemeStore.getState().theme).toBe(ThemeName.Dark);
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("setTheme to light removes the dark class", () => {
    useThemeStore.getState().setTheme(ThemeName.Dark);
    useThemeStore.getState().setTheme(ThemeName.Light);
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("toggleTheme flips between light and dark", () => {
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe(ThemeName.Dark);
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe(ThemeName.Light);
  });
});
