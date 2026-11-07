import { LockIcon } from "lucide-react";
import { useState } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { LockedReason } from "@/components/action/LockedReason";
import { hasColor, hueDotClass, hueStyle, labelPillColor } from "@/components/board/ticketTypeColor";
import { ColorPanel } from "@/components/settings/ColorPanel";
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

// One dot, not an inline swatch row, so a long list of rows keeps its colors as signal.
// The color is data first: a viewer who can't change it still sees the dot, locked beside a lock.
export const ColorSwatchButton = ({ label, value, onChange, allowNone = true, permission, projectId, verb, dotClass, pill }: ColorSwatchButtonProps) => {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(value);
  // A drag only previews; the color saves once, when the popover closes or a quick color is picked.
  const close = (color: string) => {
    setOpen(false);
    if (color !== value) onChange(color);
  };
  const action = useActionState({ permission, projectId, verb });
  const dot = (
    <span className={cn("size-2 rounded-full", dotClass ?? hueDotClass(value) ?? "border border-muted-foreground")} style={hueStyle(value)} aria-hidden />
  );
  // A colored label draws exactly as the board card's pill; an unset one stays neutral with the hollow ring, since the board's hashed fallback isn't a choice.
  const face = pill ? (
    <span
      style={hueStyle(value)}
      className={cn(
        "inline-flex h-[18px] max-w-full min-w-0 items-center gap-1.5 rounded-full px-1.5 text-micro leading-4 font-medium",
        hasColor(value) ? labelPillColor(pill, value) : "border border-border bg-foreground/[0.04] text-muted-foreground",
        "group-hover:ring-1 group-hover:ring-ring/40",
      )}
    >
      {!hasColor(value) && <span className="size-1.5 shrink-0 rounded-full border border-muted-foreground" aria-hidden />}
      <span className="truncate">{pill}</span>
    </span>
  ) : (
    dot
  );
  const pillClass = "group inline-flex h-6 max-w-full min-w-0 shrink items-center gap-1 px-0";

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
            className={cn(pill ? cn(pillClass, "text-muted-foreground") : "inline-flex h-6 items-center gap-1 rounded-md px-1.5 text-muted-foreground", "cursor-default")}
          >
            {face}
            <LockIcon aria-hidden className="size-3" />
          </button>
        </LockedReason>
      )}
      {action.state === "live" && (
        <Popover
          open={open}
          onOpenChange={(next) => {
            if (next) {
              setDraft(value);
              setOpen(true);
              return;
            }
            close(draft);
          }}
        >
          <PopoverTrigger asChild>
            <ActionButton
              permission={permission}
              projectId={projectId}
              variant="ghost"
              size={pill ? "sm" : "icon"}
              aria-label={label}
              className={pill ? cn(pillClass, "hover:bg-transparent") : "size-6"}
            >
              {face}
            </ActionButton>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-auto p-2.5" aria-label={label}>
            <ColorPanel value={draft} allowNone={allowNone} onChange={setDraft} onPick={close} />
          </PopoverContent>
        </Popover>
      )}
    </>
  );
};
