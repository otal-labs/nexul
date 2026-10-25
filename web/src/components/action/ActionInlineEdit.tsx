import { LockIcon, PencilIcon } from "lucide-react";
import type { ComponentProps } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { LockedReason } from "@/components/action/LockedReason";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ActionInlineEditProps extends Omit<ComponentProps<"button">, "aria-label">, ActionGate {
  label: string;
  onEdit?: () => void;
  // Only the live control wears it, such as a hover fill the locked one must not have.
  liveClassName?: string;
  // False for a row whose hover fill already says it edits; the locked row still fades its lock in.
  pencil?: boolean;
}

const BASE = "group/edit -mx-1 inline-flex min-w-0 items-center gap-1.5 rounded-[5px] px-1 py-0.5 text-left";
const AFFORDANCE = "size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity duration-150 group-hover/edit:opacity-100 group-focus-visible/edit:opacity-100";

// Hidden, the value still shows as plain text: it is data first and a control second. A popover trigger passes its props through to the live button.
export const ActionInlineEdit = ({ permission, projectId, answer, verb, label, onEdit, className, liveClassName, pencil = true, children, ...props }: ActionInlineEditProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  return (
    <>
      {action.state === "hidden" && <span className={cn(BASE, className)}>{children}</span>}
      {action.state === "live" && (
        <button type="button" aria-label={label} onClick={onEdit} className={cn(BASE, "hover:bg-accent/60", className, liveClassName)} {...props}>
          {children}
          {pencil && <PencilIcon aria-hidden className={AFFORDANCE} />}
        </button>
      )}
      {action.state === "locked" && (
        <LockedReason reason={action.reason} side="top">
          <button
            type="button"
            aria-label={label}
            aria-disabled
            aria-description={action.reason}
            data-locked
            onClick={(event) => event.preventDefault()}
            className={cn(BASE, "cursor-default", className)}
          >
            {children}
            <LockIcon aria-hidden className={AFFORDANCE} />
          </button>
        </LockedReason>
      )}
    </>
  );
};
