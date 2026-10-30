import { XIcon } from "lucide-react";
import type { CSSProperties } from "react";

import { cn } from "@/lib/utils";

export interface ChipOption {
  key: string;
  label: string;
  active: boolean;
  ariaLabel?: string;
  /** A label option leads with its hue dot, the same one its chip carries. */
  dotClass?: string;
  dotStyle?: CSSProperties | undefined;
  onToggle: () => void;
}

interface FilterChipRowProps {
  title?: string;
  chips: ChipOption[];
  /** A trailing Clear all, for the chip row under a toolbar. */
  onClearAll?: () => void;
}

// Active chips carry a decorative × so a selected filter reads as "on, tap
// to remove" without a separate control.
const chip = (active: boolean) =>
  cn(
    "inline-flex h-9 items-center gap-1.5 rounded-full border px-3 text-xs transition-[color,background-color,border-color] duration-150 ease-standard",
    active
      ? "border-primary bg-primary text-primary-foreground"
      : "border-border bg-card text-muted-foreground hover:border-ring/40 hover:bg-accent hover:text-foreground",
  );

export const FilterChipRow = ({ title, chips, onClearAll }: FilterChipRowProps) => (
  <div className="inline-flex flex-wrap items-center gap-x-2 gap-y-1.5">
    {title && <span className="whitespace-nowrap text-xs font-medium text-muted-foreground">{title}</span>}
    {chips.map((option) => (
      <button
        key={option.key}
        type="button"
        aria-pressed={option.active}
        aria-label={onClearAll ? `Remove ${option.label} filter` : option.ariaLabel}
        onClick={option.onToggle}
        className={chip(option.active)}
      >
        {option.dotClass && !option.active && <span className={cn("size-1.5 shrink-0 rounded-full", option.dotClass)} style={option.dotStyle} aria-hidden />}
        {option.label}
        {option.active && <XIcon className="size-3" aria-hidden />}
      </button>
    ))}
    {onClearAll && (
      <button type="button" onClick={onClearAll} className="h-9 rounded-full px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground">
        Clear all
      </button>
    )}
  </div>
);
