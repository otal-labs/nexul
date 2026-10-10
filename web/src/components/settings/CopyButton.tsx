import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useFlash } from "@/hooks/useFlash";
import { cn } from "@/lib/utils";

interface CopyButtonProps {
  /** The text to copy, or a function that fetches and copies it (a token minted on click). */
  value: string | (() => Promise<unknown>);
  /** Names what is copied: the visible label, or the accessible name of an icon-only button. */
  label: string;
  iconOnly?: boolean;
  variant?: "outline" | "ghost" | "default";
  loading?: boolean;
  className?: string;
}

// The copy icon shrinks away as a check pops in on the checkbox spring and holds for a moment; the label stays put.
export const CopyButton = ({ value, label, iconOnly = false, variant = "outline", loading = false, className }: CopyButtonProps) => {
  const [copied, flash] = useFlash();

  const copy = async () => {
    try {
      if (typeof value === "string") await navigator.clipboard.writeText(value);
      if (typeof value !== "string") await value();
      flash();
    } catch {
      // A refused clipboard leaves the text on screen to copy by hand; the fetching variant toasts its own error.
    }
  };

  return (
    <>
      <Button
        type="button"
        variant={variant}
        size={iconOnly ? "icon" : "sm"}
        loading={loading}
        aria-label={iconOnly ? label : undefined}
        title={iconOnly ? label : undefined}
        className={cn(iconOnly && "size-7 text-muted-foreground hover:text-foreground", className)}
        onClick={() => void copy()}
      >
        <span className="swap" data-icon="">
          <Copy className="size-3.5" aria-hidden {...(copied ? { "data-off": "" } : {})} />
          <Check className={cn("size-3.5", variant !== "default" && "text-success")} aria-hidden {...(copied ? {} : { "data-off": "" })} />
        </span>
        {!iconOnly && label}
      </Button>
      <span role="status" className="sr-only">
        {copied ? "Copied" : ""}
      </span>
    </>
  );
};
