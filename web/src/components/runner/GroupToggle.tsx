import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface GroupToggleProps {
  open: boolean;
  rows: number;
  name: string;
  controls: string;
  onToggle: () => void;
}

// Folds a group to its band; folded, it says how many rows it hides.
export const GroupToggle = ({ open, rows, name, controls, onToggle }: GroupToggleProps) => (
  <Button
    type="button"
    variant="ghost"
    size="sm"
    aria-expanded={open}
    aria-controls={controls}
    aria-label={`${rows} row${rows === 1 ? "" : "s"} on ${name}`}
    onClick={onToggle}
    className="gap-1 px-2 font-normal text-muted-foreground tabular-nums"
  >
    {!open && <span>{rows}</span>}
    <ChevronDownIcon aria-hidden className={cn("transition-transform duration-150 ease-standard", !open && "-rotate-90")} />
  </Button>
);
