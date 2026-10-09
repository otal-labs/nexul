import type { ReactNode } from "react";
import { Check } from "lucide-react";
import { RadioGroup as RadioGroupPrimitive } from "radix-ui";

import { cn } from "@/lib/utils";

interface ThemeTileProps {
  value: string;
  label: string;
  selected: boolean;
  /** The miniature: one ThemeMiniature, or two layered for System. */
  children: ReactNode;
}

// A radio drawn as the thing it picks: the miniature above its name, the selected one ringed in the accent with a check.
export const ThemeTile = ({ value, label, selected, children }: ThemeTileProps) => (
  <RadioGroupPrimitive.Item
    value={value}
    className="group/tile flex min-w-0 flex-col gap-2 rounded-lg p-1 text-left outline-none motion-safe:active:scale-[0.97] transition-[scale] duration-150 ease-out focus-visible:ring-[3px] focus-visible:ring-ring/40"
  >
    <span
      className={cn(
        "relative block overflow-hidden rounded-md shadow-card ring-1 transition-[box-shadow] duration-150 ease-standard",
        selected ? "ring-2 ring-brand" : "ring-border group-hover/tile:ring-foreground/25",
      )}
    >
      {children}
      {selected && (
        <span className="pop-in absolute top-1.5 right-1.5 flex size-4 items-center justify-center rounded-full bg-brand text-brand-foreground shadow-card">
          <Check className="size-2.5" strokeWidth={3} aria-hidden />
        </span>
      )}
    </span>
    <span className={cn("truncate px-0.5 text-xs", selected ? "text-foreground" : "text-muted-foreground")}>{label}</span>
  </RadioGroupPrimitive.Item>
);
