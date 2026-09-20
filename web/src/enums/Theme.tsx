export const ThemeName = {
  Light: "light",
  Dark: "dark",
} as const;

export type ThemeName = (typeof ThemeName)[keyof typeof ThemeName];

// "system" resolves to a ThemeName via matchMedia and re-resolves whenever the OS setting changes.
export const AppearanceMode = {
  System: "system",
  Light: "light",
  Dark: "dark",
} as const;

export type AppearanceMode = (typeof AppearanceMode)[keyof typeof AppearanceMode];

// Named palette, independent of light/dark; "Console" needs no override — it's what :root/.dark already paint.
export const ThemeId = {
  Console: "console",
  Ember: "ember",
  Ocean: "ocean",
  Grove: "grove",
  Clay: "clay",
  LightGreen: "light-green",
  Zen: "zen",
  Sakura: "sakura",
  Tiesen: "tiesen",
  DeepPurple: "deep-purple",
  IndigoClean: "indigo-clean",
  Brutalism: "brutalism",
} as const;

export type ThemeId = (typeof ThemeId)[keyof typeof ThemeId];
