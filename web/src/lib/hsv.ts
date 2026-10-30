export type Hsv = [hue: number, saturation: number, value: number];

export const HEX_COLOR = /^#[0-9a-f]{6}$/i;

export const hsvToHex = ([h, s, v]: Hsv): string => {
  const channel = (n: number) => {
    const k = (n + h / 60) % 6;
    return Math.round((v - v * s * Math.max(0, Math.min(k, 4 - k, 1))) * 255)
      .toString(16)
      .padStart(2, "0");
  };
  return `#${channel(5)}${channel(3)}${channel(1)}`;
};

// A grey has no hue of its own, so the caller's hue is kept and the strip doesn't jump to red.
export const hexToHsv = (hex: string, hue = 0): Hsv => {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255) as [number, number, number];
  const max = Math.max(r, g, b);
  const delta = max - Math.min(r, g, b);
  const saturation = max === 0 ? 0 : delta / max;
  if (delta === 0) return [hue, saturation, max];
  if (max === r) return [(((g - b) / delta + 6) % 6) * 60, saturation, max];
  if (max === g) return [((b - r) / delta + 2) * 60, saturation, max];
  return [((r - g) / delta + 4) * 60, saturation, max];
};
