import { ThemeId, type ThemeId as ThemeIdType } from "@/enums/Theme";

// Every role a theme may override, text-on-surface included; a role a theme leaves out keeps
// Console's zero-chroma default painted by index.css.
type Role =
  | "background"
  | "foreground"
  | "surface-2"
  | "card"
  | "card-foreground"
  | "popover"
  | "popover-foreground"
  | "secondary"
  | "secondary-foreground"
  | "muted"
  | "muted-foreground"
  | "accent"
  | "accent-foreground"
  | "primary"
  | "primary-foreground"
  | "border"
  | "input"
  | "ring";

type ThemeRoleOverrides = Partial<Record<Role, string>>;

export interface ThemeDefinition {
  id: ThemeIdType;
  label: string;
  /** Two-dot preview swatch shown on the theme library card: [light, dark]. */
  swatch: readonly [string, string];
  /** `null` for Console: it's the static default already painted by index.css. */
  colors: { light: ThemeRoleOverrides; dark: ThemeRoleOverrides } | null;
  /** Corner radius override for all four Tailwind radius steps; unset keeps the default. */
  radius?: { sm: string; md: string; lg: string; xl: string };
}

// Generated from Console's OKLCH lightness ladder and WCAG-contrast-checked; see the `color` skill for the method.
export const THEME_DEFINITIONS: readonly ThemeDefinition[] = [
  {
    id: ThemeId.Console,
    label: "Console",
    swatch: ["oklch(0.943 0 0)", "oklch(0.115 0 0)"],
    colors: null,
  },
  {
    id: ThemeId.Ember,
    label: "Ember",
    swatch: ["oklch(0.44 0.105 55)", "oklch(0.72 0.15 55)"],
    colors: {
      light: {
        background: "oklch(0.943 0.008 55)",
        "surface-2": "oklch(0.958 0.008 55)",
        card: "oklch(0.973 0.008 55)",
        popover: "oklch(0.991 0.003 55)",
        secondary: "oklch(0.922 0.008 55)",
        muted: "oklch(0.934 0.008 55)",
        border: "oklch(0.885 0.008 55)",
        input: "oklch(0.845 0.008 55)",
        accent: "oklch(0.87 0.05 55)",
        primary: "oklch(0.44 0.105 55)",
        ring: "oklch(0.44 0.105 55)",
      },
      dark: {
        background: "oklch(0.115 0.01 55)",
        "surface-2": "oklch(0.145 0.01 55)",
        card: "oklch(0.178 0.01 55)",
        popover: "oklch(0.2 0.01 55)",
        secondary: "oklch(0.218 0.01 55)",
        muted: "oklch(0.2 0.01 55)",
        border: "oklch(0.269 0.01 55)",
        input: "oklch(0.321 0.01 55)",
        accent: "oklch(0.3 0.06 55)",
        primary: "oklch(0.72 0.15 55)",
        ring: "oklch(0.72 0.15 55)",
      },
    },
  },
  {
    id: ThemeId.Ocean,
    label: "Ocean",
    swatch: ["oklch(0.44 0.075 215)", "oklch(0.72 0.125 215)"],
    colors: {
      light: {
        background: "oklch(0.943 0.008 215)",
        "surface-2": "oklch(0.958 0.008 215)",
        card: "oklch(0.973 0.008 215)",
        popover: "oklch(0.991 0.003 215)",
        secondary: "oklch(0.922 0.008 215)",
        muted: "oklch(0.934 0.008 215)",
        border: "oklch(0.885 0.008 215)",
        input: "oklch(0.845 0.008 215)",
        accent: "oklch(0.87 0.05 215)",
        primary: "oklch(0.44 0.075 215)",
        ring: "oklch(0.44 0.075 215)",
      },
      dark: {
        background: "oklch(0.115 0.01 215)",
        "surface-2": "oklch(0.145 0.01 215)",
        card: "oklch(0.178 0.01 215)",
        popover: "oklch(0.2 0.01 215)",
        secondary: "oklch(0.218 0.01 215)",
        muted: "oklch(0.2 0.01 215)",
        border: "oklch(0.269 0.01 215)",
        input: "oklch(0.321 0.01 215)",
        accent: "oklch(0.3 0.05 215)",
        primary: "oklch(0.72 0.125 215)",
        ring: "oklch(0.72 0.125 215)",
      },
    },
  },
  {
    id: ThemeId.Grove,
    label: "Grove",
    swatch: ["oklch(0.44 0.12 150)", "oklch(0.72 0.15 150)"],
    colors: {
      light: {
        background: "oklch(0.943 0.008 150)",
        "surface-2": "oklch(0.958 0.008 150)",
        card: "oklch(0.973 0.008 150)",
        popover: "oklch(0.991 0.008 150)",
        secondary: "oklch(0.922 0.008 150)",
        muted: "oklch(0.934 0.008 150)",
        border: "oklch(0.885 0.008 150)",
        input: "oklch(0.845 0.008 150)",
        accent: "oklch(0.87 0.05 150)",
        primary: "oklch(0.44 0.12 150)",
        ring: "oklch(0.44 0.12 150)",
      },
      dark: {
        background: "oklch(0.115 0.01 150)",
        "surface-2": "oklch(0.145 0.01 150)",
        card: "oklch(0.178 0.01 150)",
        popover: "oklch(0.2 0.01 150)",
        secondary: "oklch(0.218 0.01 150)",
        muted: "oklch(0.2 0.01 150)",
        border: "oklch(0.269 0.01 150)",
        input: "oklch(0.321 0.01 150)",
        accent: "oklch(0.3 0.06 150)",
        primary: "oklch(0.72 0.15 150)",
        ring: "oklch(0.72 0.15 150)",
      },
    },
  },
  {
    id: ThemeId.Clay,
    label: "Clay",
    swatch: ["oklch(0.6171 0.1375 39.04)", "oklch(0.6724 0.1308 38.76)"],
    colors: {
      light: {
        background: "oklch(0.9818 0.0054 95.1)",
        foreground: "oklch(0.3438 0.0269 95.72)",
        card: "oklch(0.9665 0.0067 97.35)",
        "card-foreground": "oklch(0.1908 0.002 106.59)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.2671 0.0196 98.94)",
        primary: "oklch(0.6171 0.1375 39.04)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.9245 0.0138 92.99)",
        "secondary-foreground": "oklch(0.4334 0.0177 98.6)",
        muted: "oklch(0.9341 0.0153 90.24)",
        "muted-foreground": "oklch(0.5341 0.0078 97.45)",
        accent: "oklch(0.9245 0.0138 92.99)",
        "accent-foreground": "oklch(0.2671 0.0196 98.94)",
        border: "oklch(0.8847 0.0069 97.36)",
        input: "oklch(0.7621 0.0156 98.35)",
        ring: "oklch(0.6171 0.1375 39.04)",
        "surface-2": "oklch(0.9663 0.008 98.88)",
      },
      dark: {
        background: "oklch(0.2679 0.0036 106.64)",
        foreground: "oklch(0.9576 0.0027 106.45)",
        card: "oklch(0.2928 0.0018 106.51)",
        "card-foreground": "oklch(0.9818 0.0054 95.1)",
        popover: "oklch(0.3085 0.0035 106.6)",
        "popover-foreground": "oklch(0.9211 0.004 106.48)",
        primary: "oklch(0.6724 0.1308 38.76)",
        "primary-foreground": "oklch(0.1908 0.002 106.59)",
        secondary: "oklch(0.9818 0.0054 95.1)",
        "secondary-foreground": "oklch(0.3085 0.0035 106.6)",
        muted: "oklch(0.2213 0.0038 106.71)",
        "muted-foreground": "oklch(0.7713 0.0169 99.07)",
        accent: "oklch(0.213 0.0078 95.42)",
        "accent-foreground": "oklch(0.9663 0.008 98.88)",
        border: "oklch(0.3618 0.0101 106.89)",
        input: "oklch(0.4336 0.0113 100.22)",
        ring: "oklch(0.6724 0.1308 38.76)",
        "surface-2": "oklch(0.2357 0.0024 67.71)",
      },
    },
  },
  {
    id: ThemeId.LightGreen,
    label: "Light Green",
    swatch: ["oklch(0.72 0.145 145.0)", "oklch(0.8871 0.2122 128.5)"],
    colors: {
      light: {
        background: "oklch(0.9892 0.0054 117.92)",
        foreground: "oklch(0.2077 0.0398 265.75)",
        card: "oklch(1.0 0.0 0.0)",
        "card-foreground": "oklch(0.2077 0.0398 265.75)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.2077 0.0398 265.75)",
        primary: "oklch(0.72 0.145 145.0)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.3717 0.0392 257.29)",
        "secondary-foreground": "oklch(0.9842 0.0034 247.86)",
        muted: "oklch(0.9683 0.0069 247.9)",
        "muted-foreground": "oklch(0.5544 0.0407 257.42)",
        accent: "oklch(0.9819 0.0181 155.83)",
        "accent-foreground": "oklch(0.4479 0.1083 151.33)",
        border: "oklch(0.9288 0.0126 255.51)",
        input: "oklch(0.9288 0.0126 255.51)",
        ring: "oklch(0.72 0.145 145.0)",
        "surface-2": "oklch(1.0 0.0 0.0)",
      },
      dark: {
        background: "oklch(0.1288 0.0406 264.69)",
        foreground: "oklch(0.9842 0.0034 247.86)",
        card: "oklch(0.2077 0.0398 265.75)",
        "card-foreground": "oklch(0.9842 0.0034 247.86)",
        popover: "oklch(0.2077 0.0398 265.75)",
        "popover-foreground": "oklch(0.9842 0.0034 247.86)",
        primary: "oklch(0.8871 0.2122 128.5)",
        "primary-foreground": "oklch(0.0 0.0 0.0)",
        secondary: "oklch(0.2795 0.0368 260.03)",
        "secondary-foreground": "oklch(0.9842 0.0034 247.86)",
        muted: "oklch(0.2795 0.0368 260.03)",
        "muted-foreground": "oklch(0.7107 0.0351 256.79)",
        accent: "oklch(0.3925 0.0896 152.53)",
        "accent-foreground": "oklch(0.8871 0.2122 128.5)",
        border: "oklch(0.2795 0.0368 260.03)",
        input: "oklch(0.2795 0.0368 260.03)",
        ring: "oklch(0.8871 0.2122 128.5)",
        "surface-2": "oklch(0.1288 0.0406 264.69)",
      },
    },
  },
  {
    id: ThemeId.Zen,
    label: "Zen",
    swatch: ["oklch(0.3012 0.0 0.0)", "oklch(0.852 0.0205 100.63)"],
    colors: {
      light: {
        background: "oklch(0.9195 0.0169 88.0)",
        foreground: "oklch(0.235 0.0 0.0)",
        card: "oklch(0.953 0.0156 86.43)",
        "card-foreground": "oklch(0.235 0.0 0.0)",
        popover: "oklch(0.953 0.0156 86.43)",
        "popover-foreground": "oklch(0.235 0.0 0.0)",
        primary: "oklch(0.3012 0.0 0.0)",
        "primary-foreground": "oklch(0.9169 0.0175 99.62)",
        secondary: "oklch(0.8647 0.0201 87.52)",
        "secondary-foreground": "oklch(0.3012 0.0 0.0)",
        muted: "oklch(0.834 0.0232 87.16)",
        "muted-foreground": "oklch(0.4688 0.0136 84.59)",
        accent: "oklch(0.9169 0.0175 99.62)",
        "accent-foreground": "oklch(0.3012 0.0 0.0)",
        border: "oklch(0.8434 0.0231 87.16)",
        input: "oklch(0.8434 0.0231 87.16)",
        ring: "oklch(0.3012 0.0 0.0)",
        "surface-2": "oklch(0.8985 0.0199 87.52)",
      },
      dark: {
        background: "oklch(0.1913 0.0 0.0)",
        foreground: "oklch(0.9173 0.0133 82.4)",
        card: "oklch(0.2264 0.0 0.0)",
        "card-foreground": "oklch(0.9173 0.0133 82.4)",
        popover: "oklch(0.2264 0.0 0.0)",
        "popover-foreground": "oklch(0.9173 0.0133 82.4)",
        primary: "oklch(0.852 0.0205 100.63)",
        "primary-foreground": "oklch(0.3329 0.0 0.0)",
        secondary: "oklch(0.252 0.0 0.0)",
        "secondary-foreground": "oklch(0.852 0.0205 100.63)",
        muted: "oklch(0.285 0.0 0.0)",
        "muted-foreground": "oklch(0.6348 0.0113 81.79)",
        accent: "oklch(0.3329 0.0 0.0)",
        "accent-foreground": "oklch(0.852 0.0205 100.63)",
        border: "oklch(0.2931 0.0 0.0)",
        input: "oklch(0.2931 0.0 0.0)",
        ring: "oklch(0.852 0.0205 100.63)",
        "surface-2": "oklch(0.173 0.0 0.0)",
      },
    },
  },
  {
    id: ThemeId.Sakura,
    label: "Sakura",
    swatch: ["oklch(0.7508 0.161 2.6)", "oklch(0.7508 0.161 2.6)"],
    colors: {
      light: {
        background: "oklch(0.9859 0.0076 48.66)",
        foreground: "oklch(0.4279 0.0265 46.62)",
        card: "oklch(1.0 0.0 0.0)",
        "card-foreground": "oklch(0.4279 0.0265 46.62)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.4279 0.0265 46.62)",
        primary: "oklch(0.7508 0.161 2.6)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.9449 0.011 54.49)",
        "secondary-foreground": "oklch(0.4279 0.0265 46.62)",
        muted: "oklch(0.9687 0.0086 44.89)",
        "muted-foreground": "oklch(0.6608 0.0272 49.58)",
        accent: "oklch(0.9239 0.0415 1.1)",
        "accent-foreground": "oklch(0.5367 0.153 7.76)",
        border: "oklch(0.9138 0.0146 50.79)",
        input: "oklch(0.9138 0.0146 50.79)",
        ring: "oklch(0.7508 0.161 2.6)",
        "surface-2": "oklch(0.9794 0.006 43.34)",
      },
      dark: {
        background: "oklch(0.1979 0.0107 39.28)",
        foreground: "oklch(0.9135 0.0123 43.27)",
        card: "oklch(0.2379 0.0124 44.53)",
        "card-foreground": "oklch(0.9135 0.0123 43.27)",
        popover: "oklch(0.2379 0.0124 44.53)",
        "popover-foreground": "oklch(0.9135 0.0123 43.27)",
        primary: "oklch(0.7508 0.161 2.6)",
        "primary-foreground": "oklch(0.1979 0.0107 39.28)",
        secondary: "oklch(0.2696 0.0148 39.27)",
        "secondary-foreground": "oklch(0.9135 0.0123 43.27)",
        muted: "oklch(0.2696 0.0148 39.27)",
        "muted-foreground": "oklch(0.6608 0.0272 49.58)",
        accent: "oklch(0.2964 0.0372 5.97)",
        "accent-foreground": "oklch(0.8436 0.0913 2.81)",
        border: "oklch(0.2937 0.0152 45.37)",
        input: "oklch(0.2937 0.0152 45.37)",
        ring: "oklch(0.7508 0.161 2.6)",
        "surface-2": "oklch(0.174 0.0094 42.99)",
      },
    },
  },
  {
    id: ThemeId.Tiesen,
    label: "Tiesen",
    swatch: ["oklch(0.5144 0.1605 267.44)", "oklch(0.5144 0.1605 267.44)"],
    colors: {
      light: {
        background: "oklch(0.9851 0.0 0.0)",
        foreground: "oklch(0.0 0.0 0.0)",
        card: "oklch(1.0 0.0 267.51)",
        "card-foreground": "oklch(0.2103 0.0 267.51)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.0 0.0 0.0)",
        primary: "oklch(0.5144 0.1605 267.44)",
        "primary-foreground": "oklch(0.97 0.014 254.6)",
        secondary: "oklch(0.94 0.0 0.0)",
        "secondary-foreground": "oklch(0.25 0.0 0.0)",
        muted: "oklch(0.97 0.0 0.0)",
        "muted-foreground": "oklch(0.44 0.0 0.0)",
        accent: "oklch(0.9214 0.0248 257.65)",
        "accent-foreground": "oklch(0.2571 0.1161 272.24)",
        border: "oklch(0.92 0.0 0.0)",
        input: "oklch(0.94 0.0 0.0)",
        ring: "oklch(0.5144 0.1605 267.44)",
        "surface-2": "oklch(1.0 0.0 267.51)",
      },
      dark: {
        background: "oklch(0.0 0.0 0.0)",
        foreground: "oklch(1.0 0.0 0.0)",
        card: "oklch(0.2103 0.0 267.51)",
        "card-foreground": "oklch(0.9461 0.0 0.0)",
        popover: "oklch(0.2103 0.0 267.51)",
        "popover-foreground": "oklch(1.0 0.0 0.0)",
        primary: "oklch(0.5144 0.1605 267.44)",
        "primary-foreground": "oklch(0.97 0.014 254.6)",
        secondary: "oklch(0.25 0.0 0.0)",
        "secondary-foreground": "oklch(0.94 0.0 0.0)",
        muted: "oklch(0.23 0.0 0.0)",
        "muted-foreground": "oklch(0.72 0.0 0.0)",
        accent: "oklch(0.32 0.0 0.0)",
        "accent-foreground": "oklch(0.9214 0.0248 257.65)",
        border: "oklch(0.26 0.0 0.0)",
        input: "oklch(0.32 0.0 0.0)",
        ring: "oklch(0.5144 0.1605 267.44)",
        "surface-2": "oklch(0.2103 0.0 267.51)",
      },
    },
  },
  {
    id: ThemeId.DeepPurple,
    label: "Deep Purple",
    swatch: ["oklch(0.4865 0.2423 291.87)", "oklch(0.6083 0.2172 297.12)"],
    colors: {
      light: {
        background: "oklch(0.9838 0.0035 247.86)",
        foreground: "oklch(0.1284 0.0267 261.59)",
        card: "oklch(1.0 0.0 0.0)",
        "card-foreground": "oklch(0.1284 0.0267 261.59)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.1284 0.0267 261.59)",
        primary: "oklch(0.4865 0.2423 291.87)",
        "primary-foreground": "oklch(0.9838 0.0035 247.86)",
        secondary: "oklch(0.9486 0.0085 303.51)",
        "secondary-foreground": "oklch(0.341 0.1625 292.95)",
        muted: "oklch(0.9679 0.0027 264.54)",
        "muted-foreground": "oklch(0.5503 0.0235 264.36)",
        accent: "oklch(0.9546 0.0227 303.29)",
        "accent-foreground": "oklch(0.4865 0.2423 291.87)",
        border: "oklch(0.9278 0.0058 264.53)",
        input: "oklch(0.9278 0.0058 264.53)",
        ring: "oklch(0.4865 0.2423 291.87)",
        "surface-2": "oklch(1.0 0.0 0.0)",
      },
      dark: {
        background: "oklch(0.1091 0.0091 301.7)",
        foreground: "oklch(0.9838 0.0035 247.86)",
        card: "oklch(0.1376 0.0118 301.06)",
        "card-foreground": "oklch(0.9838 0.0035 247.86)",
        popover: "oklch(0.1486 0.014 299.98)",
        "popover-foreground": "oklch(0.9838 0.0035 247.86)",
        primary: "oklch(0.6083 0.2172 297.12)",
        "primary-foreground": "oklch(0.1091 0.0091 301.7)",
        secondary: "oklch(0.2363 0.0582 299.64)",
        "secondary-foreground": "oklch(0.8266 0.0933 301.95)",
        muted: "oklch(0.2217 0.0242 299.7)",
        "muted-foreground": "oklch(0.7497 0.0224 301.01)",
        accent: "oklch(0.2255 0.0836 296.74)",
        "accent-foreground": "oklch(0.6083 0.2172 297.12)",
        border: "oklch(0.2505 0.0293 299.57)",
        input: "oklch(0.2505 0.0293 299.57)",
        ring: "oklch(0.6083 0.2172 297.12)",
        "surface-2": "oklch(0.1249 0.0104 301.7)",
      },
    },
  },
  {
    id: ThemeId.IndigoClean,
    label: "Indigo Clean",
    swatch: ["oklch(0.5854 0.2041 277.12)", "oklch(0.6056 0.2189 292.72)"],
    colors: {
      light: {
        background: "oklch(0.9842 0.0034 247.86)",
        foreground: "oklch(0.2077 0.0398 265.75)",
        card: "oklch(1.0 0.0 0.0)",
        "card-foreground": "oklch(0.2077 0.0398 265.75)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.2077 0.0398 265.75)",
        primary: "oklch(0.5854 0.2041 277.12)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.9288 0.0126 255.51)",
        "secondary-foreground": "oklch(0.2795 0.0368 260.03)",
        muted: "oklch(0.9683 0.0069 247.9)",
        "muted-foreground": "oklch(0.5544 0.0407 257.42)",
        accent: "oklch(0.9299 0.0334 272.79)",
        "accent-foreground": "oklch(0.3984 0.1773 277.37)",
        border: "oklch(0.9288 0.0126 255.51)",
        input: "oklch(0.9288 0.0126 255.51)",
        ring: "oklch(0.5854 0.2041 277.12)",
        "surface-2": "oklch(1.0 0.0 0.0)",
      },
      dark: {
        background: "oklch(0.2077 0.0398 265.75)",
        foreground: "oklch(0.9842 0.0034 247.86)",
        card: "oklch(0.2795 0.0368 260.03)",
        "card-foreground": "oklch(0.9842 0.0034 247.86)",
        popover: "oklch(0.2795 0.0368 260.03)",
        "popover-foreground": "oklch(0.9842 0.0034 247.86)",
        primary: "oklch(0.6056 0.2189 292.72)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.3717 0.0392 257.29)",
        "secondary-foreground": "oklch(0.9842 0.0034 247.86)",
        muted: "oklch(0.2795 0.0368 260.03)",
        "muted-foreground": "oklch(0.7107 0.0351 256.79)",
        accent: "oklch(0.3984 0.1773 277.37)",
        "accent-foreground": "oklch(0.9842 0.0034 247.86)",
        border: "oklch(0.3717 0.0392 257.29)",
        input: "oklch(0.3717 0.0392 257.29)",
        ring: "oklch(0.6056 0.2189 292.72)",
        "surface-2": "oklch(0.2795 0.0368 260.03)",
      },
    },
  },
  {
    id: ThemeId.Brutalism,
    label: "Brutalism",
    swatch: ["oklch(0.5799 0.238 29.23)", "oklch(0.628 0.2577 29.23)"],
    colors: {
      light: {
        background: "oklch(1.0 0.0 0.0)",
        foreground: "oklch(0.0 0.0 0.0)",
        card: "oklch(1.0 0.0 0.0)",
        "card-foreground": "oklch(0.0 0.0 0.0)",
        popover: "oklch(1.0 0.0 0.0)",
        "popover-foreground": "oklch(0.0 0.0 0.0)",
        primary: "oklch(0.5799 0.238 29.23)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(0.0 0.0 0.0)",
        "secondary-foreground": "oklch(1.0 0.0 0.0)",
        muted: "oklch(0.9696 0.0 0.0)",
        "muted-foreground": "oklch(0.5103 0.0 0.0)",
        accent: "oklch(0.8408 0.1725 84.2)",
        "accent-foreground": "oklch(0.0 0.0 0.0)",
        border: "oklch(0.0 0.0 0.0)",
        input: "oklch(1.0 0.0 0.0)",
        ring: "oklch(0.5799 0.238 29.23)",
        "surface-2": "oklch(0.9848 0.0 0.0)",
      },
      dark: {
        background: "oklch(0.0 0.0 0.0)",
        foreground: "oklch(1.0 0.0 0.0)",
        card: "oklch(0.1457 0.0 0.0)",
        "card-foreground": "oklch(1.0 0.0 0.0)",
        popover: "oklch(0.1457 0.0 0.0)",
        "popover-foreground": "oklch(1.0 0.0 0.0)",
        primary: "oklch(0.628 0.2577 29.23)",
        "primary-foreground": "oklch(1.0 0.0 0.0)",
        secondary: "oklch(1.0 0.0 0.0)",
        "secondary-foreground": "oklch(0.0 0.0 0.0)",
        muted: "oklch(0.2376 0.0 0.0)",
        "muted-foreground": "oklch(0.7652 0.0 0.0)",
        accent: "oklch(0.8533 0.1706 86.75)",
        "accent-foreground": "oklch(0.0 0.0 0.0)",
        border: "oklch(1.0 0.0 0.0)",
        input: "oklch(0.0 0.0 0.0)",
        ring: "oklch(0.628 0.2577 29.23)",
        "surface-2": "oklch(0.1157 0.0 0.0)",
      },
    },
    radius: { sm: "0px", md: "0px", lg: "0px", xl: "0px" },
  },
];

export const getThemeDefinition = (id: ThemeIdType): ThemeDefinition =>
  THEME_DEFINITIONS.find((t) => t.id === id) ?? THEME_DEFINITIONS[0]!;

const ROLE_CSS_VAR: Record<Role, string> = {
  background: "--background",
  foreground: "--foreground",
  "surface-2": "--surface-2",
  card: "--card",
  "card-foreground": "--card-foreground",
  popover: "--popover",
  "popover-foreground": "--popover-foreground",
  secondary: "--secondary",
  "secondary-foreground": "--secondary-foreground",
  muted: "--muted",
  "muted-foreground": "--muted-foreground",
  accent: "--accent",
  "accent-foreground": "--accent-foreground",
  primary: "--primary",
  "primary-foreground": "--primary-foreground",
  border: "--border",
  input: "--input",
  ring: "--ring",
};

const RADIUS_CSS_VAR: Record<"sm" | "md" | "lg" | "xl", string> = {
  sm: "--shape-sm",
  md: "--shape-md",
  lg: "--shape-lg",
  xl: "--shape-xl",
};

/** Paints (or clears, for Console) a theme's CSS variable overrides onto the root. */
export const applyThemePalette = (id: ThemeIdType, mode: "light" | "dark"): void => {
  const root = document.documentElement;
  const definition = getThemeDefinition(id);
  root.dataset.themeId = id === ThemeId.Console ? "" : id;
  const overrides = definition.colors?.[mode];
  for (const role of Object.keys(ROLE_CSS_VAR) as Role[]) {
    const cssVar = ROLE_CSS_VAR[role];
    const value = overrides?.[role];
    if (!value) {
      root.style.removeProperty(cssVar);
      continue;
    }
    root.style.setProperty(cssVar, value);
  }
  const radius = definition.radius;
  for (const step of Object.keys(RADIUS_CSS_VAR) as (keyof typeof RADIUS_CSS_VAR)[]) {
    const cssVar = RADIUS_CSS_VAR[step];
    const value = radius?.[step];
    if (!value) {
      root.style.removeProperty(cssVar);
      continue;
    }
    root.style.setProperty(cssVar, value);
  }
};
