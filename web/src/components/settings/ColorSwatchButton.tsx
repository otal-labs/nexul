import { LockIcon } from "lucide-react";
import { useId, useState } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { LockedReason } from "@/components/action/LockedReason";
import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { ColorPicker } from "@/components/settings/ColorPicker";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";

interface ColorSwatchButtonProps extends Pick<ActionGate, "permission" | "projectId"> {
  label: string;
  value: string;
  onChange: (color: string) => void;
  allowNone?: boolean;
  // The dot's fill when it isn't the value's own hue, such as the color a label falls back to.
  dotClass?: string;
}

// One dot, not an inline swatch row — five hues per row breaks the color-as-signal ceiling at scale.
// The color is data first: a viewer who can't change it still sees the dot, locked beside a lock.
export const ColorSwatchButton = ({ label, value, onChange, allowNone = true, permission, projectId, dotClass }: ColorSwatchButtonProps) => {
  const [open, setOpen] = useState(false);
  const id = useId();
  const action = useActionState({ permission, projectId });
  const dot = (
    <span
      className={cn("size-2.5 rounded-full", dotClass ?? (value && HUE_DOT_CLASS[value] ? HUE_DOT_CLASS[value] : "border border-dashed border-muted-foreground"))}
      aria-hidden
    />
  );

  return (
    <>
      {action.state === "hidden" && <span className="inline-flex size-6 items-center justify-center">{dot}</span>}
      {action.state === "locked" && (
        <LockedReason id={id} reason={action.reason}>
          <button
            type="button"
            aria-label={label}
            aria-disabled
            aria-describedby={id}
            data-locked
            onClick={(event) => event.preventDefault()}
            className="inline-flex h-6 cursor-default items-center gap-1 rounded-md px-1.5 text-muted-foreground"
          >
            {dot}
            <LockIcon aria-hidden className="size-3" />
          </button>
        </LockedReason>
      )}
      {action.state === "live" && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <ActionButton permission={permission} projectId={projectId} variant="ghost" size="icon" aria-label={label} className="size-6">
              {dot}
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
