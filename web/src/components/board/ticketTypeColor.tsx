// A type's icon color, 700 light and 400 dark to hold 3:1 on either canvas.
const TICKET_TYPE_COLORS: Record<string, string> = {
  task: "text-slate-700 dark:text-slate-400",
  bug: "text-fuchsia-700 dark:text-fuchsia-400",
  feature: "text-violet-700 dark:text-violet-400",
  chore: "text-amber-700 dark:text-amber-400",
  docs: "text-blue-700 dark:text-blue-400",
  documentation: "text-blue-700 dark:text-blue-400",
  epic: "text-indigo-700 dark:text-indigo-400",
  spike: "text-teal-700 dark:text-teal-400",
  design: "text-pink-700 dark:text-pink-400",
};

const TYPE_FALLBACK_PALETTE = [
  "text-cyan-700 dark:text-cyan-400",
  "text-emerald-700 dark:text-emerald-400",
  "text-orange-700 dark:text-orange-400",
  "text-fuchsia-700 dark:text-fuchsia-400",
  "text-lime-700 dark:text-lime-400",
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
  cyan: "text-cyan-700 dark:text-cyan-400",
  emerald: "text-emerald-700 dark:text-emerald-400",
  orange: "text-orange-700 dark:text-orange-400",
  fuchsia: "text-fuchsia-700 dark:text-fuchsia-400",
  lime: "text-lime-700 dark:text-lime-400",
};

export const HUE_DOT_CLASS: Record<string, string> = {
  cyan: "bg-cyan-500",
  emerald: "bg-emerald-500",
  orange: "bg-orange-500",
  fuchsia: "bg-fuchsia-500",
  lime: "bg-lime-600 dark:bg-lime-500",
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

// The board card's label pill: its hue as a 15% tint under the hue's own dark ink (4.5:1 on the card), in the same order as LABEL_DOT_PALETTE so a label keeps its colour.
const LABEL_PILL: Record<string, string> = {
  cyan: "bg-cyan-500/15 text-cyan-800 dark:text-cyan-300",
  emerald: "bg-emerald-500/15 text-emerald-800 dark:text-emerald-300",
  orange: "bg-orange-500/15 text-orange-800 dark:text-orange-300",
  fuchsia: "bg-fuchsia-500/15 text-fuchsia-800 dark:text-fuchsia-300",
  lime: "bg-lime-500/15 text-lime-900 dark:text-lime-300",
  red: "bg-red-500/15 text-red-800 dark:text-red-300",
  violet: "bg-violet-500/15 text-violet-800 dark:text-violet-300",
  amber: "bg-amber-500/15 text-amber-900 dark:text-amber-300",
  blue: "bg-blue-500/15 text-blue-800 dark:text-blue-300",
  indigo: "bg-indigo-500/15 text-indigo-800 dark:text-indigo-300",
  teal: "bg-teal-500/15 text-teal-800 dark:text-teal-300",
  pink: "bg-pink-500/15 text-pink-800 dark:text-pink-300",
};
const LABEL_PILL_ORDER = ["cyan", "emerald", "orange", "fuchsia", "lime", "red", "violet", "amber", "blue", "indigo", "teal", "pink"];

export const labelPillColor = (label: string, color?: string): string => {
  if (color && LABEL_PILL[color]) return LABEL_PILL[color];
  return LABEL_PILL[LABEL_PILL_ORDER[hashIndex(label.trim().toLowerCase(), LABEL_PILL_ORDER.length)]!]!;
};
