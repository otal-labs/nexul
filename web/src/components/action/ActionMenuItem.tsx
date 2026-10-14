import { LockIcon, type LucideIcon } from "lucide-react";
import { useId, type ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { MenuSeparator, menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ActionMenuItemProps extends ActionGate {
  children: ReactNode;
  onSelect: () => void;
  icon?: LucideIcon;
  destructive?: boolean | undefined;
  disabled?: boolean | undefined;
  // Drawn with the row, so a hidden row leaves no stray rule behind.
  separated?: boolean;
}

// Locked, the row keeps its place, loses the destructive red, and picking it does nothing, so the menu stays open.
export const ActionMenuItem = ({ permission, projectId, answer, verb, children, onSelect, icon: Icon, destructive, disabled, separated }: ActionMenuItemProps) => {
  const id = useId();
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state !== "hidden" && separated && <MenuSeparator />}
      {action.state === "live" && (
        <button type="button" className={cn(menuItemClass, destructive && menuItemDestructiveClass)} disabled={disabled} onClick={onSelect}>
          {Icon && <Icon />}
          {children}
        </button>
      )}
      {action.state === "locked" && (
        <button
          type="button"
          aria-disabled
          aria-describedby={id}
          data-locked
          onClick={(event) => event.preventDefault()}
          className={cn(
            menuItemClass,
            "cursor-default items-start text-muted-foreground hover:bg-accent/50 hover:text-muted-foreground focus-visible:bg-accent/50 focus-visible:text-muted-foreground",
          )}
        >
          {Icon && <Icon className="mt-0.5" />}
          <span className="grid flex-1 gap-0.5">
            <span>{children}</span>
            <span id={id} aria-hidden className="text-xs text-muted-foreground/90">
              {action.reason}
            </span>
          </span>
          <LockIcon aria-hidden className="mt-0.5 size-3.5" />
        </button>
      )}
    </>
  );
};
