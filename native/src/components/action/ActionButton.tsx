import type { ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedActionButton } from "@/components/action/LockedActionButton";
import { Button, type ButtonProps } from "@/components/ui/button";
import { useActionState } from "@/hooks/useActionState";

interface ActionButtonProps extends Omit<ButtonProps, "children">, ActionGate {
  children: ReactNode;
  name: string;
}

export const ActionButton = ({ permission, projectId, answer, verb, name, children, ...props }: ActionButtonProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && <Button {...props}>{children}</Button>}
      {action.state === "locked" && (
        <LockedActionButton name={name} reason={action.reason} variant={props.variant} size={props.size} className={props.className}>
          {children}
        </LockedActionButton>
      )}
    </>
  );
};
