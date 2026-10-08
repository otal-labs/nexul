// A small string hash, so a person keeps the same colours on every surface and every visit.
const hash = (seed: string): number => {
  let h = 0;
  for (const char of seed) h = (Math.imul(h, 31) + char.charCodeAt(0)) | 0;
  return h >>> 0;
};

// The face of a person without a photo: three related hues turned to a seeded angle, dark enough for white initials.
export const avatarGradient = (seed: string): string => {
  const h = hash(seed);
  const hue = h % 360;
  const turn = (h >>> 9) % 360;
  return `conic-gradient(from ${turn}deg, oklch(0.52 0.15 ${hue}), oklch(0.49 0.16 ${(hue + 50) % 360}), oklch(0.46 0.14 ${(hue + 140) % 360}), oklch(0.52 0.15 ${hue}))`;
};
