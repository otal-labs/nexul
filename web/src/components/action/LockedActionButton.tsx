import { LockIcon, type LucideIcon } from "lucide-react";

import { LockedReason } from "@/components/action/LockedReason";
import { Button, type ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Variant = NonNullable<ButtonProps["variant"]>;

const FILLED = "bg-foreground/[0.06] hover:bg-foreground/[0.06]";

const VARIANT_CLASS: Record<Variant, string> = {
  default: FILLED,
  destructive: FILLED,
  secondary: FILLED,
  outline: "bg-transparent shadow-none hover:bg-transparent",
  ghost: "hover:bg-transparent",
  link: "no-underline hover:no-underline",
};

interface LockedActionButtonProps extends Pick<ButtonProps, "variant" | "size" | "className" | "children" | "aria-label"> {
  reason: string;
  icon?: LucideIcon | undefined;
}

const LockBadge = () => (
  <span aria-hidden className="absolute right-1 bottom-1 grid size-3.5 place-items-center rounded-full bg-card ring-2 ring-card">
    <LockIcon className="size-2.5" />
  </span>
);

// aria-disabled, never disabled: it must stay in the tab order to explain itself on focus.
export const LockedActionButton = ({ variant, size, className, reason, icon: Icon, children, ...props }: LockedActionButtonProps) => {
  const glyphOnly = size === "icon";
  return (
    <LockedReason reason={reason} title={glyphOnly ? props["aria-label"] : undefined}>
      <Button
        type="button"
        variant={variant}
        size={size}
        aria-disabled
        aria-description={reason}
        data-locked
        onClick={(event) => event.preventDefault()}
        className={cn("relative cursor-default text-muted-foreground hover:text-muted-foreground motion-safe:active:scale-100", VARIANT_CLASS[variant ?? "default"], className)}
        {...props}
      >
        {!glyphOnly && <LockIcon className="size-3.5" />}
        {!glyphOnly && children}
        {glyphOnly && (Icon ? <Icon /> : children)}
        {glyphOnly && <LockBadge />}
      </Button>
    </LockedReason>
  );
};
