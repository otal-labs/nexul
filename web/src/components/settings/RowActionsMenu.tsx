import { useState } from "react";
import { MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

export interface RowAction {
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  destructive?: boolean;
}

interface RowActionsMenuProps {
  /** Row name, e.g. "Sprint 1" — becomes the trigger's accessible name. */
  subject: string;
  actions: RowAction[];
}

const itemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:bg-accent/60 disabled:pointer-events-none disabled:opacity-40";

// One menu instead of always-visible icon buttons, matching AccountMenu/BoardCreateMenu.
export const RowActionsMenu = ({ subject, actions }: RowActionsMenuProps) => {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="sm" aria-label={`Actions for ${subject}`}>
          <MoreHorizontalIcon className="size-3.5" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-40 p-1.5">
        <div className="flex flex-col gap-0.5">
          {actions.map((action) => (
            <button
              key={action.label}
              type="button"
              className={cn(
                itemClass,
                action.destructive &&
                  "text-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:bg-destructive/10 focus-visible:text-destructive",
              )}
              disabled={action.disabled}
              onClick={() => {
                setOpen(false);
                action.onSelect();
              }}
            >
              {action.label}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};
