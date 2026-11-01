import { avatarOklch, oklchToRgb } from "@nexul/client-core/avatarFill";

// React Native and SVG props take no oklch(), so the web's oklch values are converted here.
export const oklch = (l: number, c: number, h: number): string => {
  const [r, g, b] = oklchToRgb(l, c, h);
  return `rgb(${r}, ${g}, ${b})`;
};

// The web's flat avatar fill, drawn from the same shared seeded colour.
export const avatarFill = (seed: string): string => oklch(...avatarOklch(seed));

// Tokens arrive as #rrggbbaa; an SVG stop ignores that alpha, so it is split into a colour and an opacity.
export const svgPaint = (token: string | number | undefined): { color: string; opacity: number } => {
  const value = String(token ?? "#000000");
  if (!/^#[0-9a-f]{8}$/i.test(value)) return { color: value, opacity: 1 };
  return { color: value.slice(0, 7), opacity: parseInt(value.slice(7), 16) / 255 };
};
