import { create } from "zustand";
import { persist } from "zustand/middleware";

import { AppearanceMode, ThemeId, ThemeName, type AppearanceMode as AppearanceModeType, type ThemeId as ThemeIdType, type ThemeName as ThemeNameType } from "@/enums/Theme";
import { applyThemePalette } from "@/lib/themePalettes";

export const MIN_GLASS_OPACITY = 40;
export const MAX_GLASS_OPACITY = 100;
const DEFAULT_GLASS_OPACITY = 60;

export type ThemeStore = {
  theme: ThemeNameType;
  appearanceMode: AppearanceModeType;
  themeId: ThemeIdType;
  glassOpacity: number;
  setTheme: (theme: ThemeNameType) => void;
  toggleTheme: () => void;
  setAppearanceMode: (mode: AppearanceModeType) => void;
  setThemeId: (id: ThemeIdType) => void;
  setGlassOpacity: (value: number) => void;
};

const systemPrefersDark = (): boolean =>
  typeof window !== "undefined" && window.matchMedia?.("(prefers-color-scheme: dark)").matches === true;

const resolveAppearance = (mode: AppearanceModeType): ThemeNameType => {
  if (mode === AppearanceMode.System) return systemPrefersDark() ? ThemeName.Dark : ThemeName.Light;
  return mode;
};

const initialTheme = (): ThemeNameType =>
  document.documentElement.classList.contains("dark") ? ThemeName.Dark : ThemeName.Light;

const applyTheme = (theme: ThemeNameType) => {
  document.documentElement.classList.toggle("dark", theme === ThemeName.Dark);
};

const applyGlassOpacity = (value: number) => {
  document.documentElement.style.setProperty("--glass-opacity", `${value}%`);
};

export const useThemeStore = create<ThemeStore>()(
  persist(
    (set, get) => ({
      // The index.html bootstrap script already wrote the class before paint; read it back to match the DOM.
      theme: initialTheme(),
      appearanceMode: AppearanceMode.Dark,
      themeId: ThemeId.Console,
      glassOpacity: DEFAULT_GLASS_OPACITY,
      setTheme: (theme) => {
        set({ theme, appearanceMode: theme });
        applyTheme(theme);
        applyThemePalette(get().themeId, theme);
      },
      toggleTheme: () => {
        const next = get().theme === ThemeName.Dark ? ThemeName.Light : ThemeName.Dark;
        set({ theme: next, appearanceMode: next });
        applyTheme(next);
        applyThemePalette(get().themeId, next);
      },
      setAppearanceMode: (mode) => {
        const resolved = resolveAppearance(mode);
        set({ appearanceMode: mode, theme: resolved });
        applyTheme(resolved);
        applyThemePalette(get().themeId, resolved);
      },
      setThemeId: (id) => {
        set({ themeId: id });
        applyThemePalette(id, get().theme);
      },
      setGlassOpacity: (value) => {
        const clamped = Math.min(MAX_GLASS_OPACITY, Math.max(MIN_GLASS_OPACITY, value));
        set({ glassOpacity: clamped });
        applyGlassOpacity(clamped);
      },
    }),
    {
      name: "theme",
      partialize: (state) => ({
        theme: state.theme,
        appearanceMode: state.appearanceMode,
        themeId: state.themeId,
        glassOpacity: state.glassOpacity,
      }),
      onRehydrateStorage: () => (state) => {
        if (!state) return;
        applyThemePalette(state.themeId, state.theme);
        applyGlassOpacity(state.glassOpacity);
      },
    },
  ),
);

// Re-resolves on OS preference change, but only while the user has explicitly asked to follow it.
if (typeof window !== "undefined" && window.matchMedia) {
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (event) => {
    const { appearanceMode, themeId } = useThemeStore.getState();
    if (appearanceMode !== AppearanceMode.System) return;
    const resolved = event.matches ? ThemeName.Dark : ThemeName.Light;
    useThemeStore.setState({ theme: resolved });
    applyTheme(resolved);
    applyThemePalette(themeId, resolved);
  });
}
