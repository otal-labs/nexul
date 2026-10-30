import { useState } from "react";
import { MoreHorizontalIcon } from "lucide-react";

import type { PermissionAction } from "@nexul/client-core/permissionActions";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useActionStates } from "@/hooks/useActionState";
import { MenuSeparator, menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
import { cn } from "@/lib/utils";

export interface RowAction {
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  // Why a disabled action can't run, under its label: "Used by 8 tickets".
  hint?: string | undefined;
  destructive?: boolean;
  // Set when the row changes something: it is then hidden or locked for a viewer who lacks it (ADR 0152).
  permission?: PermissionAction;
  verb?: string;
}

interface RowActionsMenuProps {
  /** Row name, e.g. "Sprint 1" — becomes the trigger's accessible name. */
  subject: string;
  actions: RowAction[];
  // The project the actions write in, when the row isn't in the one in view.
  projectId?: string | undefined;
}

// One menu instead of always-visible icon buttons, matching AccountMenu/BoardCreateMenu.
export const RowActionsMenu = ({ subject, actions, projectId }: RowActionsMenuProps) => {
  const [open, setOpen] = useState(false);
  // An action with no permission is always live, so a menu holding one keeps its trigger.
  const states = useActionStates(actions.map((action) => ({ permission: action.permission ?? [], projectId, verb: action.verb })));
  if (states.every((state) => state.state === "hidden")) return null;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      {/* The popover trigger wraps the tooltip's, so the button's data-state is the menu's. */}
      <Tooltip>
        <PopoverTrigger asChild>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="sm" className="w-8 px-0" aria-label={`Actions for ${subject}`}>
              <MoreHorizontalIcon className="size-3.5" />
            </Button>
          </TooltipTrigger>
        </PopoverTrigger>
        <TooltipContent>More actions</TooltipContent>
      </Tooltip>
      <PopoverContent align="end" className="w-auto min-w-40 max-w-64 p-1">
        <div className="flex flex-col">
          {actions.map((action, i) => (
            <RowActionItem
              key={action.label}
              action={action}
              projectId={projectId}
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
  projectId: string | undefined;
  separated: boolean;
  onDone: () => void;
}

const RowActionItem = ({ action, projectId, separated, onDone }: RowActionItemProps) => {
  const select = () => {
    onDone();
    action.onSelect();
  };
  const label = (
    <span className="flex flex-col items-start">
      {action.label}
      {action.hint && <span className="text-xs font-normal text-muted-foreground">{action.hint}</span>}
    </span>
  );
  return (
    <>
      {action.permission && (
        <ActionMenuItem
          permission={action.permission}
          projectId={projectId}
          verb={action.verb}
          separated={separated}
          destructive={action.destructive}
          disabled={action.disabled}
          onSelect={select}
        >
          {label}
        </ActionMenuItem>
      )}
      {!action.permission && separated && <MenuSeparator />}
      {!action.permission && (
        <button
          type="button"
          className={cn(menuItemClass, action.destructive && menuItemDestructiveClass)}
          disabled={action.disabled}
          onClick={select}
        >
          {label}
        </button>
      )}
    </>
  );
};
