// The icon/dot color of a type or label; the Board's card pills derive their tint from it via pillClass. An unlisted type gets a stable color via hashIndex.
const TICKET_TYPE_COLORS: Record<string, string> = {
  task: "text-slate-400",
  bug: "text-red-400",
  feature: "text-violet-400",
  chore: "text-amber-400",
  docs: "text-blue-400",
  documentation: "text-blue-400",
  epic: "text-indigo-400",
  spike: "text-teal-400",
  design: "text-pink-400",
};

const TYPE_FALLBACK_PALETTE = [
  "text-cyan-400",
  "text-emerald-400",
  "text-orange-400",
  "text-fuchsia-400",
  "text-lime-400",
];

// Labels have no keyword map (freeform text), so every label hashes. Dots use 500 shade for more weight.
const LABEL_DOT_PALETTE = [
  "bg-cyan-500",
  "bg-emerald-500",
  "bg-orange-500",
  "bg-fuchsia-500",
  "bg-lime-500",
  "bg-red-500",
  "bg-violet-500",
  "bg-amber-500",
  "bg-blue-500",
  "bg-indigo-500",
  "bg-teal-500",
  "bg-pink-500",
];

const hashIndex = (value: string, length: number): number => {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % length;
};

// Shared with the backend; reused (not duplicated) by ColorPicker so swatches and badges always agree.
export const CONFIGURABLE_COLOR_NAMES = ["cyan", "emerald", "orange", "fuchsia", "lime"] as const;

export type ConfigurableColorName = (typeof CONFIGURABLE_COLOR_NAMES)[number];

const HUE_TEXT_CLASS: Record<string, string> = {
  cyan: "text-cyan-400",
  emerald: "text-emerald-400",
  orange: "text-orange-400",
  fuchsia: "text-fuchsia-400",
  lime: "text-lime-400",
};

export const HUE_DOT_CLASS: Record<string, string> = {
  cyan: "bg-cyan-500",
  emerald: "bg-emerald-500",
  orange: "bg-orange-500",
  fuchsia: "bg-fuchsia-500",
  lime: "bg-lime-500",
};

// An unrecognized `color` value falls through to the hash exactly like unset does; it can't crash the board.
export const ticketTypeColor = (typeName: string, color?: string): string => {
  if (color && HUE_TEXT_CLASS[color]) return HUE_TEXT_CLASS[color];
  const key = typeName.trim().toLowerCase();
  return TICKET_TYPE_COLORS[key] ?? TYPE_FALLBACK_PALETTE[hashIndex(key, TYPE_FALLBACK_PALETTE.length)]!;
};

export const labelDotColor = (label: string, color?: string): string => {
  if (color && HUE_DOT_CLASS[color]) return HUE_DOT_CLASS[color];
  return LABEL_DOT_PALETTE[hashIndex(label.trim().toLowerCase(), LABEL_DOT_PALETTE.length)]!;
};

// Full literal strings per hue so Tailwind emits them; the 700 text shade keeps the tint readable on the light theme.
const HUE_PILL_CLASS: Record<string, string> = {
  slate: "bg-slate-500/15 text-slate-700 dark:text-slate-300",
  red: "bg-red-500/15 text-red-700 dark:text-red-400",
  violet: "bg-violet-500/15 text-violet-700 dark:text-violet-400",
  amber: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  blue: "bg-blue-500/15 text-blue-700 dark:text-blue-400",
  indigo: "bg-indigo-500/15 text-indigo-700 dark:text-indigo-400",
  teal: "bg-teal-500/15 text-teal-700 dark:text-teal-400",
  pink: "bg-pink-500/15 text-pink-700 dark:text-pink-400",
  cyan: "bg-cyan-500/15 text-cyan-700 dark:text-cyan-400",
  emerald: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  orange: "bg-orange-500/15 text-orange-700 dark:text-orange-400",
  fuchsia: "bg-fuchsia-500/15 text-fuchsia-700 dark:text-fuchsia-400",
  lime: "bg-lime-500/15 text-lime-700 dark:text-lime-400",
};

// Tinted pill (Board cards) for an icon/dot color class such as "text-cyan-400" or "bg-cyan-500".
export const pillClass = (colorClass: string): string => {
  const hue = /^(?:text|bg)-([a-z]+)-\d+$/.exec(colorClass)?.[1] ?? "";
  return HUE_PILL_CLASS[hue] ?? HUE_PILL_CLASS.slate!;
};
