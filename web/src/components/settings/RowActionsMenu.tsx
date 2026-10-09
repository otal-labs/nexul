import { useState } from "react";
import { MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MenuSeparator, menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
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
      {/* The popover trigger wraps the tooltip's, so the button's data-state is the menu's. */}
      <Tooltip>
        <PopoverTrigger asChild>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm" aria-label={`Actions for ${subject}`}>
              <MoreHorizontalIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
        </PopoverTrigger>
        <TooltipContent>More actions</TooltipContent>
      </Tooltip>
      <PopoverContent align="end" className="w-40 p-1">
        <div className="flex flex-col">
          {actions.map((action, i) => (
            <RowActionItem
              key={action.label}
              action={action}
              // Destructive actions sit apart, after a rule.
              separated={action.destructive === true && i > 0 && !actions[i - 1]?.destructive}
              onDone={() => setOpen(false)}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

interface RowActionItemProps {
  action: RowAction;
  separated: boolean;
  onDone: () => void;
}

const RowActionItem = ({ action, separated, onDone }: RowActionItemProps) => (
  <>
    {separated && <MenuSeparator />}
    <button
      type="button"
      className={cn(menuItemClass, action.destructive && menuItemDestructiveClass)}
      disabled={action.disabled}
      onClick={() => {
        onDone();
        action.onSelect();
      }}
    >
      {action.label}
    </button>
  </>
);
