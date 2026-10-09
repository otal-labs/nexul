import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type SettingsStatusTone = "success" | "warning" | "destructive" | "info" | "muted";

const dotClass: Record<SettingsStatusTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  destructive: "bg-destructive",
  info: "bg-info",
  muted: "bg-muted-foreground/50",
};

interface SettingsStatusProps {
  tone: SettingsStatusTone;
  children: ReactNode;
  /** A muted second part after a dot, such as who connected it and when. */
  detail?: ReactNode;
  className?: string;
}

// A setting's state at a glance: a status dot beside plain text, never a tinted chip. A changed tone crossfades the dot.
export const SettingsStatus = ({ tone, children, detail, className }: SettingsStatusProps) => (
  <span className={cn("inline-flex min-w-0 items-center gap-1.5 text-xs", className)}>
    <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full transition-colors duration-150 ease-standard", dotClass[tone])} />
    <span className="shrink-0 font-medium text-foreground">{children}</span>
    {detail && <span className="min-w-0 truncate text-muted-foreground">· {detail}</span>}
  </span>
);
