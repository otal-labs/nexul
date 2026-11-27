import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type SettingsStatusTone = "success" | "warning" | "destructive" | "info" | "muted";

export const statusDotClass: Record<SettingsStatusTone, string> = {
  success: "bg-success",
  warning: "bg-warning-mark",
  destructive: "bg-destructive",
  info: "bg-info",
  muted: "bg-muted-foreground/50",
};

interface SettingsStatusProps {
  tone: SettingsStatusTone;
  children: ReactNode;
  /** A muted second part 16px after the state, such as who connected it and when. */
  detail?: ReactNode;
  className?: string;
  /** Colours the detail when it is the part that needs attention, such as how long a computer has been gone. */
  detailClassName?: string;
}

// A setting's state at a glance: a status dot beside plain text, never a tinted chip. A changed tone crossfades the dot.
export const SettingsStatus = ({ tone, children, detail, className, detailClassName }: SettingsStatusProps) => (
  <span className={cn("inline-flex min-w-0 items-center gap-1.5 text-xs", className)}>
    <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full transition-colors duration-150 ease-standard", statusDotClass[tone])} />
    <span className={cn("shrink-0 font-medium", tone === "muted" ? "text-muted-foreground" : "text-foreground")}>{children}</span>
    {/* Space alone sets the detail apart; the hidden comma keeps it a separate phrase when read aloud. */}
    {detail && (
      <span className={cn("ml-2.5 min-w-0 truncate text-muted-foreground", detailClassName)}>
        <span className="sr-only">, </span>
        {detail}
      </span>
    )}
  </span>
);
