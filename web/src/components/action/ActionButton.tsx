import type { LucideIcon } from "lucide-react";
import { Link } from "react-router";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedActionButton } from "@/components/action/LockedActionButton";
import { Button, type ButtonProps } from "@/components/ui/button";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

export interface ActionButtonProps extends Omit<ButtonProps, "asChild">, ActionGate {
  // A locked labelled button swaps it for the lock; an icon button keeps it beside the lock.
  icon?: LucideIcon;
  // Only the live control wears it: a locked destructive control loses its red.
  liveClassName?: string;
  // A second permission the control also needs: the first refusal wins, so a link that runs two writes locks on either.
  alsoNeeds?: ActionGate;
  // A control that goes to a page stays a link while live, so it opens in a new tab and reads as one.
  to?: string;
}

export const ActionButton = ({ permission, projectId, answer, verb, alsoNeeds, icon: Icon, liveClassName, to, children, ...props }: ActionButtonProps) => {
  const first = useActionState({ permission, projectId, answer, verb });
  const second = useActionState(alsoNeeds ?? { permission, projectId, answer, verb });
  const action = first.state !== "live" ? first : second;
  return (
    <>
      {action.state === "live" && !to && (
        <Button {...props} className={cn(props.className, liveClassName)}>
          {Icon && <Icon />}
          {children}
        </Button>
      )}
      {action.state === "live" && to && (
        <Button {...props} asChild className={cn(props.className, liveClassName)}>
          <Link to={to}>
            {Icon && <Icon />}
            {children}
          </Link>
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
