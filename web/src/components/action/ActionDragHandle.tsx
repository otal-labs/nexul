import { GripVerticalIcon } from "lucide-react";
import { useId, type ComponentProps } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedReason } from "@/components/action/LockedReason";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ActionDragHandleProps extends Omit<ComponentProps<"button">, "aria-label">, ActionGate {
  label: string;
}

// Locked, the sortable's listeners are dropped, so nothing can start a drag.
export const ActionDragHandle = ({ permission, projectId, answer, verb, label, className, ...props }: ActionDragHandleProps) => {
  const id = useId();
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "live" && (
        <button type="button" aria-label={label} className={cn("cursor-grab touch-none rounded-md p-0.5 text-muted-foreground/60 hover:text-foreground active:cursor-grabbing", className)} {...props}>
          <GripVerticalIcon aria-hidden className="size-3.5" />
        </button>
      )}
      {action.state === "locked" && (
        <LockedReason id={id} reason={action.reason} side="top">
          <button
            type="button"
            aria-label={label}
            aria-disabled
            aria-describedby={id}
            data-locked
            onClick={(event) => event.preventDefault()}
            className={cn("cursor-default rounded-md p-0.5 text-muted-foreground/45", className)}
          >
            <GripVerticalIcon aria-hidden className="size-3.5" />
          </button>
        </LockedReason>
      )}
    </>
  );
};
