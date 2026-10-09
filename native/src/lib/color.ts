// A small string hash, so a person or project keeps the same colours on every surface, as on the web.
const hash = (seed: string): number => {
  let h = 0;
  for (const char of seed) h = (Math.imul(h, 31) + char.charCodeAt(0)) | 0;
  return h >>> 0;
};

const toSrgb = (linear: number) => {
  const v = linear <= 0.0031308 ? 12.92 * linear : 1.055 * linear ** (1 / 2.4) - 0.055;
  return Math.round(Math.min(1, Math.max(0, v)) * 255);
};

// React Native and SVG props take no oklch(), so the web's oklch stops are converted here.
export const oklch = (l: number, c: number, h: number): string => {
  const a = c * Math.cos((h * Math.PI) / 180);
  const b = c * Math.sin((h * Math.PI) / 180);
  const l_ = (l + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m_ = (l - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s_ = (l - 0.0894841775 * a - 1.291485548 * b) ** 3;
  const r = 4.0767416621 * l_ - 3.3077115913 * m_ + 0.2309699292 * s_;
  const g = -1.2684380046 * l_ + 2.6097574011 * m_ - 0.3413193965 * s_;
  const bl = -0.0041960863 * l_ - 0.7034186147 * m_ + 1.707614701 * s_;
  return `rgb(${toSrgb(r)}, ${toSrgb(g)}, ${toSrgb(bl)})`;
};

export interface AvatarGradient {
  // The gradient's angle in degrees and its three stops, dark enough for white initials.
  turn: number;
  stops: [string, string, string];
}

// The web's conic avatar as a linear sweep of the same three seeded hues (SVG on a phone has no conic gradient).
export const avatarGradient = (seed: string): AvatarGradient => {
  const h = hash(seed);
  const hue = h % 360;
  return {
    turn: (h >>> 9) % 360,
    stops: [oklch(0.52, 0.15, hue), oklch(0.49, 0.16, (hue + 50) % 360), oklch(0.46, 0.14, (hue + 140) % 360)],
  };
};

// Tokens arrive as #rrggbbaa; an SVG stop ignores that alpha, so it is split into a colour and an opacity.
export const svgPaint = (token: string | number | undefined): { color: string; opacity: number } => {
  const value = String(token ?? "#000000");
  if (!/^#[0-9a-f]{8}$/i.test(value)) return { color: value, opacity: 1 };
  return { color: value.slice(0, 7), opacity: parseInt(value.slice(7), 16) / 255 };
};
