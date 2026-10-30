import { LockIcon } from "lucide-react";
import { useState } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { LockedReason } from "@/components/action/LockedReason";
import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ColorSwatchButtonProps extends Pick<ActionGate, "permission" | "projectId" | "verb"> {
  label: string;
  value: string;
  onChange: (color: string) => void;
  allowNone?: boolean;
  // The dot's fill when it isn't the value's own hue, such as the color a label falls back to.
  dotClass?: string;
  // Draws the button as the label's own pill, its dot leading the text, so the label is what you click to recolour it.
  pill?: string;
}

// One dot, not an inline swatch row — five hues per row breaks the color-as-signal ceiling at scale.
// The color is data first: a viewer who can't change it still sees the dot, locked beside a lock.
export const ColorSwatchButton = ({ label, value, onChange, allowNone = true, permission, projectId, verb, dotClass, pill }: ColorSwatchButtonProps) => {
  const [open, setOpen] = useState(false);
  const action = useActionState({ permission, projectId, verb });
  const dot = (
    <span
      className={cn(pill ? "relative -top-px size-1.5 shrink-0 self-center" : "size-2", "rounded-full", dotClass ?? (value && HUE_DOT_CLASS[value] ? HUE_DOT_CLASS[value] : "border border-muted-foreground"))}
      aria-hidden
    />
  );
  const face = (
    <>
      {dot}
      {pill && <span className="truncate">{pill}</span>}
    </>
  );
  const pillClass = "inline-flex h-6 max-w-full min-w-0 shrink items-baseline gap-1.5 py-0.5 leading-[1.125rem] rounded-full border border-border bg-foreground/[0.04] px-2 text-xs font-medium text-muted-foreground";

  return (
    <>
      {action.state === "hidden" && <span className={cn(pill ? pillClass : "inline-flex size-6 items-center justify-center")}>{face}</span>}
      {action.state === "locked" && (
        <LockedReason reason={action.reason}>
          <button
            type="button"
            aria-label={label}
            aria-disabled
            aria-description={action.reason}
            data-locked
            onClick={(event) => event.preventDefault()}
            className={cn(pill ? pillClass : "inline-flex h-6 items-center gap-1 rounded-md px-1.5 text-muted-foreground", "cursor-default")}
          >
            {face}
            <LockIcon aria-hidden className="size-3" />
          </button>
        </LockedReason>
      )}
      {action.state === "live" && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <ActionButton
              permission={permission}
              projectId={projectId}
              variant="ghost"
              size={pill ? "sm" : "icon"}
              aria-label={label}
              className={pill ? cn(pillClass, "hover:bg-accent hover:text-foreground") : "size-6"}
            >
              {face}
            </ActionButton>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-auto p-1.5">
            <ColorPicker
              label={label}
              value={value}
              allowNone={allowNone}
              onChange={(color) => {
                setOpen(false);
                onChange(color);
              }}
            />
          </PopoverContent>
        </Popover>
      )}
    </>
  );
};
