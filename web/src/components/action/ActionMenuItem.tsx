import { LockIcon, type LucideIcon } from "lucide-react";
import { useId, type ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { MenuSeparator, menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
import type { MenuParts } from "@/components/listpane/rowMenu";
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
  // Inside a Radix menu the row is the menu's own item, so it closes the menu and joins its arrow keys.
  ui?: MenuParts;
}

const LOCKED_CLASS = "cursor-default items-start text-muted-foreground";
const LOCKED_HOVER =
  "hover:bg-accent/50 hover:text-muted-foreground focus-visible:bg-accent/50 focus-visible:text-muted-foreground data-[highlighted]:bg-accent/50 data-[highlighted]:text-muted-foreground focus:bg-accent/50 focus:text-muted-foreground";

// Locked, the row keeps its place, loses the destructive red, and picking it does nothing, so the menu stays open.
export const ActionMenuItem = ({ permission, projectId, answer, verb, children, onSelect, icon: Icon, destructive, disabled, separated, ui }: ActionMenuItemProps) => {
  const id = useId();
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state !== "hidden" && separated && <MenuSeparator />}
      {action.state === "live" && !ui && (
        <button type="button" className={cn(menuItemClass, destructive && menuItemDestructiveClass)} disabled={disabled} onClick={onSelect}>
          {Icon && <Icon />}
          {children}
        </button>
      )}
      {action.state === "live" && ui && (
        <ui.Item variant={destructive ? "destructive" : "default"} onSelect={onSelect}>
          {Icon && <Icon />}
          {children}
        </ui.Item>
      )}
      {action.state === "locked" && !ui && (
        <button
          type="button"
          aria-disabled
          aria-describedby={id}
          data-locked
          onClick={(event) => event.preventDefault()}
          className={cn(menuItemClass, LOCKED_CLASS, LOCKED_HOVER)}
        >
          <LockedBody id={id} icon={Icon} reason={action.reason}>
            {children}
          </LockedBody>
        </button>
      )}
      {action.state === "locked" && ui && (
        <ui.Item aria-disabled aria-describedby={id} data-locked onSelect={(event) => event.preventDefault()} className={cn(LOCKED_CLASS, LOCKED_HOVER)}>
          <LockedBody id={id} icon={Icon} reason={action.reason}>
            {children}
          </LockedBody>
        </ui.Item>
      )}
    </>
  );
};

interface LockedBodyProps {
  id: string;
  icon: LucideIcon | undefined;
  reason: string;
  children: ReactNode;
}

const LockedBody = ({ id, icon: Icon, reason, children }: LockedBodyProps) => (
  <>
    {Icon && <Icon className="mt-0.5" />}
    <span className="grid flex-1 gap-0.5">
      <span>{children}</span>
      <span id={id} aria-hidden className="text-xs text-muted-foreground/90">
        {reason}
      </span>
    </span>
    <LockIcon aria-hidden className="mt-0.5 size-3.5" />
  </>
);
