// A type's icon color on the card and the ticket screen: 700 in light and 400 in dark hold 3:1 on either canvas. The words
// stay muted; status and labels never fill a chip either, a label keeps its hue on a dot.
const HUE_ICON_CLASS: Record<string, string> = {
  slate: "text-slate-700 dark:text-slate-400",
  red: "text-red-700 dark:text-red-400",
  violet: "text-violet-700 dark:text-violet-400",
  amber: "text-amber-700 dark:text-amber-400",
  blue: "text-blue-700 dark:text-blue-400",
  indigo: "text-indigo-700 dark:text-indigo-400",
  teal: "text-teal-700 dark:text-teal-400",
  pink: "text-pink-700 dark:text-pink-400",
  cyan: "text-cyan-700 dark:text-cyan-400",
  emerald: "text-emerald-700 dark:text-emerald-400",
  orange: "text-orange-700 dark:text-orange-400",
  fuchsia: "text-fuchsia-700 dark:text-fuchsia-400",
  lime: "text-lime-700 dark:text-lime-400",
};

const TYPE_HUES: Record<string, string> = {
  task: "slate",
  bug: "red",
  feature: "violet",
  chore: "amber",
  docs: "blue",
  documentation: "blue",
  epic: "indigo",
  spike: "teal",
  design: "pink",
};

const TYPE_FALLBACK_HUES = ["cyan", "emerald", "orange", "fuchsia", "lime"];

const LABEL_DOTS = [
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

const CONFIGURABLE_HUES = new Set(["cyan", "emerald", "orange", "fuchsia", "lime"]);

const hashIndex = (value: string, length: number): number => {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % length;
};

// An unrecognized `color` value falls through to the hash exactly like unset does; it can't crash the card.
export const ticketTypeColor = (typeName: string, color?: string): string => {
  if (color && CONFIGURABLE_HUES.has(color)) return HUE_ICON_CLASS[color]!;
  const key = typeName.trim().toLowerCase();
  return HUE_ICON_CLASS[TYPE_HUES[key] ?? TYPE_FALLBACK_HUES[hashIndex(key, TYPE_FALLBACK_HUES.length)]!]!;
};

export const labelDot = (label: string): string => LABEL_DOTS[hashIndex(label.trim().toLowerCase(), LABEL_DOTS.length)]!;
