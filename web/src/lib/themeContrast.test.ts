import { beforeEach, describe, expect, it } from "vitest";

import { ThemeId } from "@/enums/Theme";
import { currentRowFillClass } from "@/components/CurrentRow";
import css from "@/index.css?raw";
import { applyThemePalette, THEME_DEFINITIONS } from "@/lib/themePalettes";

type Rgb = [number, number, number];
type Mode = "light" | "dark";

const PANEL_OPACITY = 0.85;
const MIN_RATIO = 4.5;
const DEFAULT_MUTED_RATIO = 6;
const MIN_EDGE_RATIO = 1.1;

const block = (selector: string): string => css.match(new RegExp(`\\n${selector} \\{([\\s\\S]*?)\\n\\}`))?.[1] ?? "";
const DEFAULTS: Record<Mode, string> = { light: block(":root"), dark: block("\\.dark") };

const cssValue = (mode: Mode, name: string): string => {
  const value = DEFAULTS[mode].match(new RegExp(`--${name}:\\s*([^;]+);`))?.[1]?.trim();
  if (!value) throw new Error(`index.css has no --${name} in ${mode}`);
  return value;
};

const rootValue = (mode: Mode, name: string): string => {
  const value = document.documentElement.style.getPropertyValue(`--${name}`) || cssValue(mode, name);
  const alias = value.match(/^var\(--([\w-]+)\)$/);
  return alias ? rootValue(mode, alias[1]!) : value;
};

const oklab = (value: string): Rgb => {
  const parts = value.match(/^oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)/);
  if (!parts) throw new Error(`not an oklch colour: ${value}`);
  const [l, c, h] = [Number(parts[1]), Number(parts[2]), (Number(parts[3]) * Math.PI) / 180];
  return [l, c * Math.cos(h), c * Math.sin(h)];
};

// A hairline written with alpha (the default palette's) lands on whatever surface it is drawn over.
const alphaOf = (value: string): number => Number(value.match(/\/\s*([\d.]+)\s*\)$/)?.[1] ?? 1);

const mixOklab = (a: Rgb, b: Rgb, share: number): Rgb => a.map((v, i) => v * share + b[i]! * (1 - share)) as Rgb;

const encode = (linear: number): number => (linear <= 0.0031308 ? 12.92 * linear : 1.055 * linear ** (1 / 2.4) - 0.055);
const decode = (channel: number): number => (channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4);

// Out-of-gamut values clip to the sRGB edge, as the browser does on an sRGB display.
const toSrgb = ([l, a, b]: Rgb): Rgb => {
  const [lc, mc, sc] = [(l + 0.3963377774 * a + 0.2158037573 * b) ** 3, (l - 0.1055613458 * a - 0.0638541728 * b) ** 3, (l - 0.0894841775 * a - 1.291485548 * b) ** 3];
  const linear = [4.0767416621 * lc - 3.3077115913 * mc + 0.2309699292 * sc, -1.2684380046 * lc + 2.6097574011 * mc - 0.3413193965 * sc, -0.0041960863 * lc - 0.7034186147 * mc + 1.707614701 * sc];
  return linear.map((v) => encode(Math.min(1, Math.max(0, v)))) as Rgb;
};

const luminance = ([r, g, b]: Rgb): number => 0.2126 * decode(r) + 0.7152 * decode(g) + 0.0722 * decode(b);

const ratio = (a: Rgb, b: Rgb): number => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi! + 0.05) / (lo! + 0.05);
};

const surfaces = (mode: Mode): Record<string, Rgb> => {
  const background = oklab(rootValue(mode, "background"));
  const card = oklab(rootValue(mode, "card"));
  const panelFill = mode === "dark" ? mixOklab(card, background, 0.6) : card;
  const canvas = toSrgb(background);
  const panel = toSrgb(panelFill).map((v, i) => v * PANEL_OPACITY + canvas[i]! * (1 - PANEL_OPACITY)) as Rgb;
  return {
    background: canvas,
    "surface-2": toSrgb(oklab(rootValue(mode, "surface-2"))),
    card: toSrgb(card),
    popover: toSrgb(oklab(rootValue(mode, "popover"))),
    panel,
  };
};

const ink = (mode: Mode, name: string): Rgb => toSrgb(oklab(rootValue(mode, name)));

// The current row's brand tint as the shipped class spells it: `bg-brand/12`, with a `dark:` override.
const tintShare = (mode: Mode): number => {
  const light = Number(currentRowFillClass.match(/(?:^|\s)bg-brand\/(\d+)/)?.[1]);
  const dark = Number(currentRowFillClass.match(/dark:bg-brand\/(\d+)/)?.[1] ?? light);
  return (mode === "dark" ? dark : light) / 100;
};

const withTint = (mode: Mode, surface: Rgb): Rgb => {
  const share = tintShare(mode);
  return ink(mode, "brand").map((v, i) => v * share + surface[i]! * (1 - share)) as Rgb;
};

// Every pair a palette paints text on: body ink and secondary ink on each surface, ink on the current row's brand tint over each, and the label on the brand fill.
const failingPairs = (mode: Mode, mutedFloor: number): string[] => {
  const onSurface = surfaces(mode);
  const pairs: [string, number, number][] = [];
  for (const [surface, rgb] of Object.entries(onSurface)) {
    pairs.push([`foreground on ${surface}`, ratio(ink(mode, "foreground"), rgb), MIN_RATIO]);
    pairs.push([`muted-foreground on ${surface}`, ratio(ink(mode, "muted-foreground"), rgb), mutedFloor]);
  }
  pairs.push(["card-foreground on card", ratio(ink(mode, "card-foreground"), onSurface.card!), MIN_RATIO]);
  pairs.push(["popover-foreground on popover", ratio(ink(mode, "popover-foreground"), onSurface.popover!), MIN_RATIO]);
  for (const [surface, rgb] of Object.entries(onSurface)) {
    pairs.push([`foreground on the current row over ${surface}`, ratio(ink(mode, "foreground"), withTint(mode, rgb)), MIN_RATIO]);
  }
  pairs.push(["brand-foreground on brand", ratio(ink(mode, "brand-foreground"), ink(mode, "brand")), MIN_RATIO]);
  return pairs.filter(([, value, floor]) => value < floor).map(([pair, value, floor]) => `${pair}: ${value.toFixed(2)} < ${floor}`);
};

// The field's edge over the surfaces a field sits on: a hairline by design, but never so close to its surface that the field vanishes.
const failingEdges = (mode: Mode): string[] => {
  const value = rootValue(mode, "input");
  const edge = toSrgb(oklab(value));
  const share = alphaOf(value);
  const { card, panel } = surfaces(mode);
  return Object.entries({ card, panel })
    .map(([surface, rgb]) => [surface, ratio(edge.map((v, i) => v * share + rgb![i]! * (1 - share)) as Rgb, rgb!)] as const)
    .filter(([, value]) => value < MIN_EDGE_RATIO)
    .map(([surface, value]) => `input border on ${surface}: ${value.toFixed(2)} < ${MIN_EDGE_RATIO}`);
};

describe("palette text contrast", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("style");
  });

  it.each(THEME_DEFINITIONS.flatMap((theme) => (["light", "dark"] as const).map((mode) => ({ id: theme.id, label: theme.label, mode }))))(
    "$label $mode keeps every text pair at 4.5:1 (the default palette's muted text at 6:1)",
    ({ id, mode }) => {
      applyThemePalette(id, mode);
      expect(failingPairs(mode, id === ThemeId.Console ? DEFAULT_MUTED_RATIO : MIN_RATIO)).toEqual([]);
    },
  );
});

describe("palette field edge", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("style");
  });

  it.each(THEME_DEFINITIONS.flatMap((theme) => (["light", "dark"] as const).map((mode) => ({ id: theme.id, label: theme.label, mode }))))(
    "$label $mode keeps the input border visible (1.1:1) against its surface",
    ({ id, mode }) => {
      applyThemePalette(id, mode);
      expect(failingEdges(mode)).toEqual([]);
    },
  );
});
