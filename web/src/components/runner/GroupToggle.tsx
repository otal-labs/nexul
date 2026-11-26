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

// Leads its band, before the name it folds; the rows it hides are in its label.
export const GroupToggle = ({ open, rows, name, controls, onToggle }: GroupToggleProps) => (
  <Button
    type="button"
    variant="ghost"
    size="icon"
    aria-expanded={open}
    aria-controls={controls}
    aria-label={`${rows} row${rows === 1 ? "" : "s"} on ${name}`}
    onClick={onToggle}
    className="size-6 rounded-sm text-muted-foreground"
  >
    <ChevronDownIcon aria-hidden className={cn("transition-transform duration-150 ease-standard", !open && "-rotate-90")} />
  </Button>
);
