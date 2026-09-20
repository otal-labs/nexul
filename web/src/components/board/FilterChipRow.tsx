import { XIcon } from "lucide-react";

import { cn } from "@/lib/utils";

export interface ChipOption {
  key: string;
  label: string;
  active: boolean;
  ariaLabel?: string;
  onToggle: () => void;
}

interface FilterChipRowProps {
  title: string;
  chips: ChipOption[];
}

// Active chips carry a decorative × so a selected filter reads as "on, tap
// to remove" without a separate control.
const chip = (active: boolean) =>
  cn(
    "inline-flex h-9 items-center gap-1 rounded-full border px-3 text-xs transition-[color,background-color,border-color] duration-150 ease-standard",
    active
      ? "border-primary bg-primary text-primary-foreground"
      : "border-border bg-card text-muted-foreground hover:border-ring/40 hover:bg-accent hover:text-foreground",
  );

export const FilterChipRow = ({ title, chips }: FilterChipRowProps) => (
  <div className="inline-flex flex-wrap items-center gap-x-2 gap-y-1.5">
    <span className="whitespace-nowrap text-xs font-medium text-muted-foreground">{title}</span>
    {chips.map((option) => (
      <button
        key={option.key}
        type="button"
        aria-pressed={option.active}
        aria-label={option.ariaLabel}
        onClick={option.onToggle}
        className={chip(option.active)}
      >
        {option.label}
        {option.active && <XIcon className="size-3" aria-hidden />}
      </button>
    ))}
  </div>
);
