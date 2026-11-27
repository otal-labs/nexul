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
  // Names the switch by other text on the page, such as the row's sentence; wins over the label.
  labelledBy?: string;
  title?: string;
  className?: string;
}

// A switch shows state too, so a viewer who can't flip it still sees it, disabled.
export const ActionSwitch = ({ permission, projectId, answer, verb, checked, onCheckedChange, size = "default", disabled = false, title, className, "aria-label": label, labelledBy }: ActionSwitchProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && (
        <Switch size={size} checked={checked} disabled={disabled} onCheckedChange={onCheckedChange} aria-label={label} {...(labelledBy && { "aria-labelledby": labelledBy })} className={className} {...(title && { title })} />
      )}
      {action.state === "hidden" && (
        <Switch size={size} checked={checked} disabled aria-label={label} {...(labelledBy && { "aria-labelledby": labelledBy })} className={className} {...(title && { title })} />
      )}
      {action.state === "locked" && (
        // The tooltip's trigger is a wrapper: on the switch itself its data-state would replace checked and unchecked.
        <LockedReason reason={action.reason} side="top">
          <span className="inline-flex">
            <Switch
              size={size}
              checked={checked}
              aria-label={label} {...(labelledBy && { "aria-labelledby": labelledBy })}
              aria-disabled
              aria-description={action.reason}
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
