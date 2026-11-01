// A small string hash, so a person keeps the same colour on every surface and every visit.
const hash = (seed: string): number => {
  let h = 0;
  for (const char of seed) h = (Math.imul(h, 31) + char.charCodeAt(0)) | 0;
  return h >>> 0;
};

// Sequential logins ("bob_a", "bob_b") differ by one, so the hash is scrambled before it picks a slot.
const scramble = (h: number): number => {
  let x = h ^ Math.imul(15, 0x9e3779b1);
  x = Math.imul(x ^ (x >>> 16), 0x85ebca6b);
  x = Math.imul(x ^ (x >>> 13), 0xc2b2ae35);
  return (x ^ (x >>> 16)) >>> 0;
};

const LIGHTNESS = 0.5;
const CHROMA = 0.11;
const SLOTS = 12;
// Stepping the wheel by 150 degrees (coprime with the 12 slots) puts neighbouring slots far apart.
const SLOT_STEP = 150;

// The face of a person without a photo: one of 12 evenly spaced hues at a fixed lightness and chroma that white initials clear 4.5:1 on.
export const avatarOklch = (seed: string): [lightness: number, chroma: number, hue: number] => [
  LIGHTNESS,
  CHROMA,
  ((scramble(hash(seed)) % SLOTS) * SLOT_STEP) % 360,
];

export const avatarFill = (seed: string): string => `oklch(${avatarOklch(seed).join(" ")})`;

const toSrgb = (linear: number) => {
  const v = linear <= 0.0031308 ? 12.92 * linear : 1.055 * linear ** (1 / 2.4) - 0.055;
  return Math.round(Math.min(1, Math.max(0, v)) * 255);
};

// An oklch colour's 0-255 sRGB channels, clipped as a browser does, for a consumer that takes no oklch().
export const oklchToRgb = (l: number, c: number, h: number): [r: number, g: number, b: number] => {
  const a = c * Math.cos((h * Math.PI) / 180);
  const b = c * Math.sin((h * Math.PI) / 180);
  const l_ = (l + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m_ = (l - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s_ = (l - 0.0894841775 * a - 1.291485548 * b) ** 3;
  return [
    toSrgb(4.0767416621 * l_ - 3.3077115913 * m_ + 0.2309699292 * s_),
    toSrgb(-1.2684380046 * l_ + 2.6097574011 * m_ - 0.3413193965 * s_),
    toSrgb(-0.0041960863 * l_ - 0.7034186147 * m_ + 1.707614701 * s_),
  ];
};

// The same fill as #rrggbb: the collab cursor plugin accepts only hex, and appends an alpha to it for a selection.
export const avatarFillHex = (seed: string): string =>
  `#${oklchToRgb(...avatarOklch(seed))
    .map((v) => v.toString(16).padStart(2, "0"))
    .join("")}`;
