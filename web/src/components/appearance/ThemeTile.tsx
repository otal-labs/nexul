import type { ReactNode } from "react";
import { RadioGroup as RadioGroupPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

interface ThemeTileProps {
  value: string;
  label: string;
  selected: boolean;
  /** The miniature: one ThemeMiniature, or two layered for System. */
  children: ReactNode;
}

// Mode and Theme share one column grid, so a mode tile is the size of a theme tile and the two read as one picker.
export const pickerGridClass = "grid grid-cols-3 gap-3 @lg:grid-cols-4";

// A radio drawn as the thing it picks: the miniature above its name, the selected one ringed in the accent and named in ink.
export const ThemeTile = ({ value, label, selected, children }: ThemeTileProps) => (
  <RadioGroupPrimitive.Item
    value={value}
    className="group/tile flex min-w-0 flex-col gap-2 rounded-lg p-1 text-left outline-none motion-safe:active:scale-[0.97] transition-[scale] duration-150 ease-out"
  >
    <span
      className={cn(
        "relative block overflow-hidden rounded-md shadow-card ring-1 transition-[box-shadow] duration-150 ease-standard",
        selected ? "ring-2 ring-brand" : "ring-border group-hover/tile:ring-foreground/25",
      )}
    >
      {children}
    </span>
    <span className={cn("truncate px-0.5 text-xs", selected ? "font-medium text-foreground" : "text-muted-foreground")}>{label}</span>
  </RadioGroupPrimitive.Item>
);
