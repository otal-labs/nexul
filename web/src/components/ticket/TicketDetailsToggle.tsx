import { ChevronDownIcon } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface TicketDetailsToggleProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The id of the rail it folds. */
  controls: string;
  className?: string;
}

// Under a 52rem panel the rail folds behind this row, so the body starts under the header strip.
export const TicketDetailsToggle = ({ open, onOpenChange, controls, className }: TicketDetailsToggleProps) => (
  <button
    type="button"
    aria-expanded={open}
    aria-controls={controls}
    onClick={() => onOpenChange(!open)}
    className={cn(
      microheaderClass,
      "-mx-2 mb-2 flex h-8 items-center gap-1.5 justify-self-start rounded-md px-2 transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground",
      className,
    )}
  >
    Details
    <ChevronDownIcon
      className={cn("size-3.5 transition-transform duration-150 ease-standard", open && "rotate-180")}
      aria-hidden
    />
  </button>
);
