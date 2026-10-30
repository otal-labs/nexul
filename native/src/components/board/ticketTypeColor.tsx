// A type's icon color on the card and the ticket screen: 700 in light and 400 in dark hold 3:1 on either canvas. The words
// stay muted; status and labels never fill a chip either, a label keeps its hue on a dot.
const HUE_ICON_CLASS: Record<string, string> = {
  slate: "text-slate-700 dark:text-slate-400",
  rose: "text-rose-700 dark:text-rose-400",
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
  bug: "fuchsia",
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

const HUE_DOT_CLASS: Record<string, string> = {
  cyan: "bg-cyan-500",
  emerald: "bg-emerald-500",
  orange: "bg-orange-500",
  fuchsia: "bg-fuchsia-500",
  lime: "bg-lime-600 dark:bg-lime-500",
};

const CONFIGURABLE_HUES = new Set(["cyan", "emerald", "orange", "fuchsia", "lime"]);

const hashIndex = (value: string, length: number): number => {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % length;
};

// An owner's own hex has no light and dark shade, so the icon takes it as its color prop, as picked.
const hexColor = (color?: string): string | undefined => (color && /^#[0-9a-f]{6}$/i.test(color) ? color : undefined);

export const hexIconColor = (color?: string): { color?: string } => {
  const hex = hexColor(color);
  return hex ? { color: hex } : {};
};

// An unrecognized `color` value falls through to the hash exactly like unset does; it can't crash the card.
export const ticketTypeColor = (typeName: string, color?: string): string => {
  if (hexColor(color)) return "";
  if (color && CONFIGURABLE_HUES.has(color)) return HUE_ICON_CLASS[color]!;
  const key = typeName.trim().toLowerCase();
  return HUE_ICON_CLASS[TYPE_HUES[key] ?? TYPE_FALLBACK_HUES[hashIndex(key, TYPE_FALLBACK_HUES.length)]!]!;
};

// A label's configured hue or hex wins; unset hashes its name onto a hue, the same as the web.
export const labelDot = (label: string, color?: string): string => {
  if (hexColor(color)) return "";
  if (color && HUE_DOT_CLASS[color]) return HUE_DOT_CLASS[color];
  return LABEL_DOTS[hashIndex(label.trim().toLowerCase(), LABEL_DOTS.length)]!;
};

export const labelDotStyle = (color?: string): { backgroundColor: string } | undefined => {
  const hex = hexColor(color);
  return hex ? { backgroundColor: hex } : undefined;
};
