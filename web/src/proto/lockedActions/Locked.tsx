import { useId, useState, type ComponentProps, type ReactNode } from "react";
import { GripVerticalIcon, LockIcon, PencilIcon, type LucideIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ContextMenuItem } from "@/components/ui/context-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

// a: quiet lock + tooltip, b: press for why, c: reason spelled out under the control.
export type Variant = "a" | "b" | "c";
export type Kind = "primary" | "outline" | "icon";

const stop = (event: { preventDefault: () => void }) => event.preventDefault();

// A and C: the control keeps its size and shape, drops its fill and hue, and reads in muted ink (6:1, never opacity).
const quietClass = (kind: Kind) =>
  cn(
    "relative inline-flex shrink-0 cursor-default items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap text-muted-foreground select-none",
    "[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
    kind === "primary" && "h-9 bg-foreground/[0.06] px-3",
    kind === "outline" && "h-9 border border-border px-3",
    kind === "icon" && "size-9",
  );

// B: the control keeps its look at half strength, like disabled, but stays pressable: pressing explains.
const LockedControl = ({ variant, kind, ...props }: ComponentProps<"button"> & { variant: Variant; kind: Kind }) => (
  <>
    {variant === "b" && (
      <Button
        {...props}
        variant={kind === "primary" ? "default" : kind === "outline" ? "outline" : "ghost"}
        size={kind === "icon" ? "icon" : "default"}
        className="relative opacity-50 hover:opacity-60"
      />
    )}
    {variant !== "b" && <button {...props} className={quietClass(kind)} />}
  </>
);

const IconLockBadge = () => (
  <span className="absolute right-1 bottom-1 grid size-3.5 place-items-center rounded-full bg-card ring-2 ring-card" aria-hidden>
    <LockIcon className="size-2.5" />
  </span>
);

const ButtonFace = ({ kind, label, icon: Icon }: { kind: Kind; label: string; icon: LucideIcon }) => (
  <>
    {kind !== "icon" && <LockIcon className="size-3.5" />}
    {kind !== "icon" && label}
    {kind === "icon" && <Icon />}
    {kind === "icon" && <IconLockBadge />}
  </>
);

const ReasonBubble = ({ label, reason }: { label: string; reason: string }) => (
  <div className="flex gap-2.5">
    <LockIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
    <div className="grid gap-0.5">
      <p className="text-sm font-medium">{label} is locked</p>
      <p className="text-sm text-muted-foreground">{reason}</p>
    </div>
  </div>
);

interface LockedButtonProps {
  variant: Variant;
  kind: Kind;
  label: string;
  icon: LucideIcon;
  reason: string;
  touch?: boolean;
}

export const LockedButton = ({ variant, kind, label, icon, reason, touch = false }: LockedButtonProps) => {
  const id = useId();
  const common = {
    type: "button" as const,
    "aria-disabled": true,
    "aria-describedby": id,
    "aria-label": kind === "icon" ? label : undefined,
    "data-locked": "",
  };
  const pressForWhy = variant === "b" || (touch && (variant === "a" || kind === "icon"));
  const lineUnder = variant === "c" && kind !== "icon";
  const hoverTip = !pressForWhy && !lineUnder;
  const face = <ButtonFace kind={kind} label={label} icon={icon} />;

  return (
    <>
      {pressForWhy && (
        <Popover>
          <PopoverTrigger asChild>
            <LockedControl {...common} variant={variant} kind={kind}>
              {face}
            </LockedControl>
          </PopoverTrigger>
          <PopoverContent side="bottom" align="end" className="w-auto max-w-72">
            <ReasonBubble label={label} reason={reason} />
          </PopoverContent>
        </Popover>
      )}
      {hoverTip && (
        <Tooltip>
          <TooltipTrigger asChild>
            <LockedControl {...common} onClick={stop} variant={variant} kind={kind}>
              {face}
            </LockedControl>
          </TooltipTrigger>
          <TooltipContent side="bottom" className="max-w-60 text-left">
            {kind === "icon" && <span className="block font-medium text-foreground">{label}</span>}
            {reason}
          </TooltipContent>
        </Tooltip>
      )}
      {lineUnder && (
        <span className="inline-flex flex-col items-end gap-1">
          <LockedControl {...common} onClick={stop} variant={variant} kind={kind}>
            {face}
          </LockedControl>
          <span className="text-xs whitespace-nowrap text-muted-foreground" aria-hidden>
            {reason}
          </span>
        </span>
      )}
      <span id={id} hidden>
        {reason}
      </span>
    </>
  );
};

interface LockedMenuItemProps {
  variant: Variant;
  label: string;
  icon: LucideIcon;
  reason: string;
}

// A and C: the reason is the item's second line. B: the item explains itself when picked, and the menu stays open.
export const LockedMenuItem = ({ variant, label, icon: Icon, reason }: LockedMenuItemProps) => {
  const id = useId();
  const [open, setOpen] = useState(false);
  const showLine = variant !== "b" || open;
  const select = (event: Event) => {
    event.preventDefault();
    if (variant === "b") setOpen(true);
  };
  return (
    <ContextMenuItem
      aria-disabled
      aria-describedby={id}
      onSelect={select}
      className="items-start text-muted-foreground focus:bg-accent/50 focus:text-muted-foreground"
    >
      <Icon className="mt-0.5" />
      <span className="grid flex-1 gap-0.5">
        <span>{label}</span>
        {showLine && (
          <span id={id} className="text-xs text-muted-foreground/90">
            {reason}
          </span>
        )}
        {!showLine && (
          <span id={id} hidden>
            {reason}
          </span>
        )}
      </span>
      <LockIcon className="mt-1 size-3.5" />
    </ContextMenuItem>
  );
};

interface ReasonTriggerProps {
  variant: Variant;
  label: string;
  reason: string;
  touch?: boolean;
  className: string;
  children: ReactNode;
}

// The small locked affordances (a drag handle, an inline edit) carry the reason the way their variant's buttons do; C's
// card line explains them, so they carry nothing of their own.
const ReasonTrigger = ({ variant, label, reason, touch = false, className, children }: ReasonTriggerProps) => {
  const id = useId();
  const pressForWhy = variant === "b" || (touch && variant === "a");
  const common = { type: "button" as const, "aria-disabled": true, "aria-describedby": id, "aria-label": label, className };
  return (
    <>
      {pressForWhy && (
        <Popover>
          <PopoverTrigger asChild>
            <button {...common}>{children}</button>
          </PopoverTrigger>
          <PopoverContent side="bottom" align="start" className="w-auto max-w-72">
            <ReasonBubble label={label} reason={reason} />
          </PopoverContent>
        </Popover>
      )}
      {!pressForWhy && variant === "a" && (
        <Tooltip>
          <TooltipTrigger asChild>
            <button {...common} onClick={stop}>
              {children}
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">{reason}</TooltipContent>
        </Tooltip>
      )}
      {!pressForWhy && variant === "c" && (
        <button {...common} onClick={stop}>
          {children}
        </button>
      )}
      <span id={id} hidden>
        {reason}
      </span>
    </>
  );
};

export const LockedHandle = (props: { variant: Variant; name: string; reason: string; touch?: boolean }) => (
  <ReasonTrigger
    {...props}
    label={`Move ${props.name}`}
    className={cn(
      "grid h-7 w-5 cursor-default place-items-center rounded-[5px] text-muted-foreground/45",
      props.variant === "b" && "opacity-60",
    )}
  >
    <GripVerticalIcon className="size-4" />
  </ReasonTrigger>
);

export const LockedInlineName = (props: { variant: Variant; name: string; reason: string; touch?: boolean }) => (
  <ReasonTrigger
    {...props}
    label={`Rename ${props.name}`}
    className="group/name -mx-1 inline-flex min-w-0 cursor-default items-center gap-1.5 rounded-[5px] px-1 py-0.5 text-left text-sm font-medium text-foreground"
  >
    <span className="truncate">{props.name}</span>
    {props.variant !== "c" && (
      <LockIcon className="size-3.5 text-muted-foreground opacity-0 transition-opacity duration-150 group-hover/name:opacity-100 group-focus-visible/name:opacity-100" />
    )}
  </ReasonTrigger>
);

export const OpenHandle = ({ name }: { name: string }) => (
  <button type="button" aria-label={`Move ${name}`} className="grid h-7 w-5 cursor-grab place-items-center rounded-[5px] text-muted-foreground hover:bg-accent hover:text-foreground">
    <GripVerticalIcon className="size-4" />
  </button>
);

export const OpenInlineName = ({ name }: { name: string }) => (
  <button type="button" className="group/name -mx-1 inline-flex min-w-0 items-center gap-1.5 rounded-[5px] px-1 py-0.5 text-left text-sm font-medium hover:bg-accent/60">
    <span className="truncate">{name}</span>
    <PencilIcon className="size-3.5 text-muted-foreground opacity-0 group-hover/name:opacity-100" />
  </button>
);
