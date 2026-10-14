import type { LucideIcon } from "lucide-react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedActionButton } from "@/components/action/LockedActionButton";
import { Button, type ButtonProps } from "@/components/ui/button";
import { useActionState } from "@/hooks/useActionState";

export interface ActionButtonProps extends Omit<ButtonProps, "asChild">, ActionGate {
  // A locked labelled button swaps it for the lock; an icon button keeps it beside the lock.
  icon?: LucideIcon;
}

export const ActionButton = ({ permission, projectId, answer, verb, icon: Icon, children, ...props }: ActionButtonProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && (
        <Button {...props}>
          {Icon && <Icon />}
          {children}
        </Button>
      )}
      {action.state === "locked" && (
        <LockedActionButton
          reason={action.reason}
          icon={Icon}
          variant={props.variant}
          size={props.size}
          className={props.className}
          aria-label={props["aria-label"]}
        >
          {children}
        </LockedActionButton>
      )}
    </>
  );
};
