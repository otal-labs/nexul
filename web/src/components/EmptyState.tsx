import { useId, type ComponentType, type ReactNode } from "react";
import { Inbox } from "lucide-react";

import { displayTitleClass } from "@/components/PageHeader";
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

type Icon = ComponentType<{ className?: string }>;

// One mark for every empty: the icon on two orbits in the light field's colours, ember and blue on opposite sides.
const OrbitMark = ({ icon: Icon, compact }: { icon: Icon; compact: boolean }) => {
  const id = useId();
  return (
    <span className={cn("relative grid place-items-center", compact ? "size-14" : "size-28")}>
      <svg aria-hidden viewBox="0 0 112 112" className="absolute inset-0 size-full" fill="none">
        <defs>
          <linearGradient id={id} x1="0" y1="1" x2="1" y2="0">
            <stop offset="0" stopColor="oklch(from var(--field-cool) l c h)" />
            <stop offset="0.55" stopColor="oklch(from var(--field-pink) l c h)" />
            <stop offset="1" stopColor="var(--brand)" />
          </linearGradient>
        </defs>
        <circle cx="56" cy="56" r="54" stroke={`url(#${id})`} strokeOpacity="0.45" />
        <circle cx="56" cy="56" r="40" stroke={`url(#${id})`} strokeOpacity="0.85" strokeDasharray="2 5" strokeLinecap="round" />
        <circle cx="94.2" cy="39.8" r="3" fill="var(--brand)" />
        <circle cx="21.8" cy="78" r="2.5" fill="oklch(from var(--field-cool) l c h)" />
      </svg>
      <span className={cn("relative grid place-items-center rounded-full bg-card shadow-card ring-1 ring-border", compact ? "size-7" : "size-12")}>
        <Icon className={cn("text-muted-foreground", compact ? "size-3.5" : "size-5")} aria-hidden />
      </span>
    </span>
  );
};

export const EmptyState = ({
  icon = Inbox,
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
        "animate-in fade-in-0 slide-in-from-bottom-1 flex flex-col items-center text-center duration-200 ease-out",
        compact ? "gap-2 p-5" : "px-6 py-14",
        className,
      )}
    >
      <OrbitMark icon={icon} compact={compact} />
      {compact && <h2 className="text-sm font-medium text-foreground">{title}</h2>}
      {!compact && <h2 className={cn(displayTitleClass, "mt-6 max-w-md text-[1.625rem] text-foreground")}>{title}</h2>}
      {message && (
        <p className={cn("max-w-sm text-pretty text-muted-foreground", compact ? "text-xs" : "mt-2 text-sm")}>{message}</p>
      )}
      {action && <div className={compact ? "mt-1" : "mt-6"}>{action}</div>}
    </div>
  );
};
