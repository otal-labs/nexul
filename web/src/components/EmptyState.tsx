import type { ComponentType, ReactNode } from "react";
import { Inbox } from "lucide-react";

import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  message?: string;
  action?: ReactNode;
  role?: string;
  className?: string;
  /** "compact" for a stacked sub-section on a detail page; "default" (full padding) for a standalone list page. */
  size?: "default" | "compact";
}

export const EmptyState = ({
  icon: Icon = Inbox,
  title,
  message,
  action,
  role,
  className,
  size = "default",
}: EmptyStateProps) => {
  const compact = size === "compact";
  return (
    <div
      role={role}
      className={cn(
        "animate-in fade-in-0 slide-in-from-bottom-1 flex flex-col items-center rounded-lg border border-dashed text-center text-muted-foreground duration-150 ease-out",
        compact ? "gap-2 p-5" : "gap-3 p-10",
        className,
      )}
    >
      <span
        className={cn(
          "flex items-center justify-center rounded-lg border border-border bg-muted/60 shadow-card",
          compact ? "size-8" : "size-11",
        )}
      >
        <Icon className={compact ? "size-4 text-primary" : "size-5 text-primary"} aria-hidden />
      </span>
      <h2 className={cn("font-semibold tracking-tight text-foreground", compact ? "text-sm" : "text-lg")}>
        {title}
      </h2>
      {message && (
        <p className={cn("max-w-sm text-muted-foreground", compact ? "text-xs" : "text-sm")}>{message}</p>
      )}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
};
