// Dot color for a ticket's type or label — a compact indicator on the board row, per the Mono Console's
// "colored dot next to plain text" rule (no tinted-pill exception here; that's a web board-card-only case).
// An unlisted type/label gets a stable color via hashIndex, matching web's ticketTypeColor.tsx palette.
const TYPE_DOT_COLORS: Record<string, string> = {
  task: "bg-slate-400",
  bug: "bg-red-400",
  feature: "bg-violet-400",
  chore: "bg-amber-400",
  docs: "bg-blue-400",
  documentation: "bg-blue-400",
  epic: "bg-indigo-400",
  spike: "bg-teal-400",
  design: "bg-pink-400",
};

const DOT_FALLBACK_PALETTE = [
  "bg-cyan-400",
  "bg-emerald-400",
  "bg-orange-400",
  "bg-fuchsia-400",
  "bg-lime-400",
  "bg-red-400",
  "bg-violet-400",
  "bg-amber-400",
  "bg-blue-400",
  "bg-indigo-400",
  "bg-teal-400",
  "bg-pink-400",
];

const HUE_DOT_CLASS: Record<string, string> = {
  cyan: "bg-cyan-400",
  emerald: "bg-emerald-400",
  orange: "bg-orange-400",
  fuchsia: "bg-fuchsia-400",
  lime: "bg-lime-400",
};

const hashIndex = (value: string, length: number): number => {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) | 0;
  }
  return Math.abs(hash) % length;
};

// An unrecognized `color` value falls through to the hash exactly like unset does; it can't crash the row.
export const ticketTypeDotColor = (typeName: string, color?: string): string => {
  if (color && HUE_DOT_CLASS[color]) return HUE_DOT_CLASS[color];
  const key = typeName.trim().toLowerCase();
  return TYPE_DOT_COLORS[key] ?? DOT_FALLBACK_PALETTE[hashIndex(key, DOT_FALLBACK_PALETTE.length)]!;
};

export const labelDotColor = (label: string): string =>
  DOT_FALLBACK_PALETTE[hashIndex(label.trim().toLowerCase(), DOT_FALLBACK_PALETTE.length)]!;
