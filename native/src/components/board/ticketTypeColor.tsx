// The type pill, the web board card's: a 15% tint of the hue with an 800 (light) or 400 (dark) text shade, which
// holds 4.5:1 in both themes. Status and labels never use a tinted pill; a label keeps its hue on a dot.
const HUE_PILL_CLASS: Record<string, string> = {
  slate: "bg-slate-500/15 text-slate-800 dark:text-slate-300",
  red: "bg-red-500/15 text-red-800 dark:text-red-400",
  violet: "bg-violet-500/15 text-violet-800 dark:text-violet-400",
  amber: "bg-amber-500/15 text-amber-800 dark:text-amber-400",
  blue: "bg-blue-500/15 text-blue-800 dark:text-blue-400",
  indigo: "bg-indigo-500/15 text-indigo-800 dark:text-indigo-400",
  teal: "bg-teal-500/15 text-teal-800 dark:text-teal-400",
  pink: "bg-pink-500/15 text-pink-800 dark:text-pink-400",
  cyan: "bg-cyan-500/15 text-cyan-800 dark:text-cyan-400",
  emerald: "bg-emerald-500/15 text-emerald-800 dark:text-emerald-400",
  orange: "bg-orange-500/15 text-orange-800 dark:text-orange-400",
  fuchsia: "bg-fuchsia-500/15 text-fuchsia-800 dark:text-fuchsia-400",
  lime: "bg-lime-500/15 text-lime-800 dark:text-lime-400",
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
export const ticketTypePill = (typeName: string, color?: string): string => {
  if (color && CONFIGURABLE_HUES.has(color)) return HUE_PILL_CLASS[color]!;
  const key = typeName.trim().toLowerCase();
  return HUE_PILL_CLASS[TYPE_HUES[key] ?? TYPE_FALLBACK_HUES[hashIndex(key, TYPE_FALLBACK_HUES.length)]!]!;
};

export const labelDot = (label: string): string => LABEL_DOTS[hashIndex(label.trim().toLowerCase(), LABEL_DOTS.length)]!;
