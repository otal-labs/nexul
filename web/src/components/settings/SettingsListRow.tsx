import type { ComponentProps, ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsListRowProps extends Omit<ComponentProps<"li">, "children"> {
  // The 16px slot before the name: a colour dot, the hollow ring, a status icon or a mark.
  lead?: ReactNode;
  name: ReactNode;
  // A muted line on the name's own line: a template state, a machine, a repository's role.
  meta?: ReactNode;
  // Right-aligned mono, saying what it counts: "8 tickets".
  count?: ReactNode;
  // The row's `⋯`: shown on hover or focus where the device can hover, always where it can't.
  actions?: ReactNode;
  // Keeps the actions in view at rest, for a row whose action is its point (a Deploy).
  showActions?: boolean;
}

// One row for every project settings list: the board's grammar, a hairline apart, lifting on hover inside its rules.
export const SettingsListRow = ({ lead, name, meta, count, actions, showActions = false, className, ...props }: SettingsListRowProps) => (
  <li
    className={cn(
      "group/row relative flex min-h-11 items-center gap-2 text-sm px-2 before:pointer-events-none before:absolute before:inset-x-0 before:inset-y-1 before:rounded-md before:transition-colors before:duration-150 before:ease-standard hover:before:bg-accent/40",
      className,
    )}
    {...props}
  >
    {lead !== undefined && <span className="relative flex size-4 shrink-0 items-center justify-center">{lead}</span>}
    {/* Name, meta and count share one baseline; centred as boxes, the 12px mono count sits a pixel above the name. */}
    <span className="relative flex min-w-0 flex-1 items-baseline gap-2">
      {/* Both give way when the row is short, the name three times faster, so a short meta stays readable; the count stays whole. */}
      <span className="flex min-w-0 shrink-[3]">{name}</span>
      {meta && <span className="min-w-0 truncate text-xs text-muted-foreground">{meta}</span>}
      {count && <span className="ml-auto shrink-0 pl-2 font-mono text-xs text-muted-foreground tabular-nums">{count}</span>}
    </span>
    {actions && (
      <span
        className={cn(
          "relative flex shrink-0",
          !showActions &&
            "[@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:has-data-[state=open]:opacity-100",
        )}
      >
        {actions}
      </span>
    )}
  </li>
);

// The leading slot's colour: a configured hue's dot, or the hollow ring that says no colour is set.
export const ColorDot = ({ className }: { className?: string | undefined }) => (
  <span className={cn("size-2 rounded-full", className ?? "border border-muted-foreground")} aria-hidden />
);
