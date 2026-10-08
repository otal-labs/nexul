import { useState } from "react";
import { MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
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
      <PopoverContent align="end" className="w-40 p-1">
        <div className="flex flex-col">
          {actions.map((action) => (
            <button
              key={action.label}
              type="button"
              className={cn(menuItemClass, action.destructive && menuItemDestructiveClass)}
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
