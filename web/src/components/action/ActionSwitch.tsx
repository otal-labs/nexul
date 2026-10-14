import { useId } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedReason } from "@/components/action/LockedReason";
import { Switch } from "@/components/ui/switch";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ActionSwitchProps extends ActionGate {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  size?: "sm" | "default";
  disabled?: boolean;
  "aria-label": string;
  title?: string;
  className?: string;
}

// A switch shows state too, so a viewer who can't flip it still sees it, disabled.
export const ActionSwitch = ({ permission, projectId, answer, verb, checked, onCheckedChange, size = "default", disabled = false, title, className, "aria-label": label }: ActionSwitchProps) => {
  const id = useId();
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && (
        <Switch size={size} checked={checked} disabled={disabled} onCheckedChange={onCheckedChange} aria-label={label} className={className} {...(title && { title })} />
      )}
      {action.state === "hidden" && (
        <Switch size={size} checked={checked} disabled aria-label={label} className={className} {...(title && { title })} />
      )}
      {action.state === "locked" && (
        // The tooltip's trigger is a wrapper: on the switch itself its data-state would replace checked and unchecked.
        <LockedReason id={id} reason={action.reason} side="top">
          <span className="inline-flex">
            <Switch
              size={size}
              checked={checked}
              aria-label={label}
              aria-disabled
              aria-describedby={id}
              data-locked
              onCheckedChange={() => undefined}
              className={cn("cursor-default opacity-60", className)}
            />
          </span>
        </LockedReason>
      )}
    </>
  );
};
