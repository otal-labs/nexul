import type { ReactNode, SyntheticEvent } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Microheader } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface OverflowCountProps {
  // What the count stands for, in words: "2 more labels". It heads the list and opens the trigger's accessible name.
  label: string;
  // The items the count hides, listed when it opens.
  items: string[];
  // What the trigger draws; "+N" by default.
  children?: ReactNode;
  className?: string;
}

// A row or card that opens on click and drags on press must not do either from the count or its portalled list.
const keepFromParent = (event: SyntheticEvent) => event.stopPropagation();
// No pointerdown: the board drags on its mouse and touch sensors, as CardPerson isolates its picker.
const isolation = { onClick: keepFromParent, onMouseDown: keepFromParent, onTouchStart: keepFromParent, onKeyDown: keepFromParent };

// A "+N" that names what it hides on tap or click, since a touch screen never shows a title.
export const OverflowCount = ({ label, items, children, className }: OverflowCountProps) => (
  <Popover>
    <PopoverTrigger
      type="button"
      aria-label={`${label}: ${items.join(", ")}`}
      {...isolation}
      className={cn(
        "hit-area pointer-events-auto shrink-0 cursor-pointer rounded-sm font-mono tabular-nums hover:text-foreground hover:underline hover:underline-offset-2 data-[state=open]:text-foreground",
        className,
      )}
    >
      {children ?? `+${items.length}`}
    </PopoverTrigger>
    <PopoverContent className="w-auto max-w-xs p-2" {...isolation}>
      <Microheader className="px-1 pb-1">{label}</Microheader>
      <ul className="max-h-64 overflow-y-auto text-xs">
        {items.map((item, index) => (
          <li key={index} dir="auto" className="px-1 py-0.5 break-words">
            {item}
          </li>
        ))}
      </ul>
    </PopoverContent>
  </Popover>
);
