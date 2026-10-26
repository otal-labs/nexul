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
