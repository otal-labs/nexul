import type { ComponentProps, HTMLAttributes, ReactNode, Ref } from "react";

import { cn } from "@/lib/utils";

interface SettingsListRowProps extends Omit<ComponentProps<"li">, "children"> {
  // The 16px slot before the name: a colour dot, the hollow ring, a status icon or a mark.
  lead?: ReactNode;
  name: ReactNode;
  // A muted line on the name's own line: a template state, a machine, a repository's role.
  meta?: ReactNode;
  // Right-aligned mono, saying what it counts: "8 tickets".
  count?: ReactNode;
  // An always-visible control before the actions, such as a switch.
  control?: ReactNode;
  // The row's `⋯`: shown on hover or focus where the device can hover, always where it can't.
  actions?: ReactNode;
  // Keeps the actions in view at rest, for a row whose action is its point (a Deploy).
  showActions?: boolean;
  // Makes everything but the actions the drag handle of a sortable list.
  handleRef?: Ref<HTMLSpanElement>;
  handleProps?: HTMLAttributes<HTMLSpanElement> | undefined;
}

// Where the device can hover, a row's actions take the count's place instead of holding an empty column after it.
const hoverOnly = "[@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:group-has-[[data-state=open]]/row:opacity-100";
const givesWay = "[@media(hover:hover)]:group-focus-within/row:opacity-0 [@media(hover:hover)]:group-hover/row:opacity-0 [@media(hover:hover)]:group-has-[[data-state=open]]/row:opacity-0";

// One row for every project settings list: the board's grammar, a hairline apart, its text on the card's edge and its hover lift reaching 8px past it.
export const SettingsListRow = ({ lead, name, meta, count, control, actions, showActions = false, handleRef, handleProps, className, ...props }: SettingsListRowProps) => {
  const overlay = !!count && !!actions && !showActions;
  return (
    <li
      className={cn(
        "group/row relative flex min-h-11 items-center gap-2 text-sm before:pointer-events-none before:absolute before:-inset-x-2 before:inset-y-1 before:rounded-md before:transition-colors before:duration-150 before:ease-standard hover:before:bg-accent/40",
        className,
      )}
      {...props}
    >
      <span className="relative flex min-w-0 flex-1 items-center gap-2 self-stretch">
        <span
          ref={handleRef}
          {...handleProps}
          className={cn(
            "relative flex min-w-0 flex-1 items-center gap-2 self-stretch rounded-md",
            handleProps && "cursor-grab select-none active:cursor-grabbing",
          )}
        >
          {lead !== undefined && <span className="flex size-4 shrink-0 items-center justify-center">{lead}</span>}
          {/* Name, meta and count share one baseline; centred as boxes, the 12px mono count sits a pixel above the name. */}
          <span className="flex min-w-0 flex-1 items-baseline gap-2">
            {/* The meta keeps its width up to half the line and the count stays whole; the name truncates in what is left. */}
            <span className="flex min-w-0">{name}</span>
            {meta && <span className="max-w-1/2 shrink-0 truncate text-xs text-muted-foreground">{meta}</span>}
            {count && (
              <span className={cn("ml-auto shrink-0 pl-2 font-mono text-xs text-muted-foreground tabular-nums transition-opacity duration-150 ease-standard", overlay && givesWay)}>
                {count}
              </span>
            )}
          </span>
        </span>
        {actions && (
          <span
            className={cn(
              "relative flex shrink-0 items-center transition-opacity duration-150 ease-standard",
              !showActions && hoverOnly,
              overlay && "[@media(hover:hover)]:absolute [@media(hover:hover)]:inset-y-0 [@media(hover:hover)]:right-0",
            )}
          >
            {actions}
          </span>
        )}
      </span>
      {control && <span className="relative flex shrink-0 items-center">{control}</span>}
    </li>
  );
};

// The leading slot's colour: a configured hue's dot, or the hollow ring that says no colour is set.
export const ColorDot = ({ className }: { className?: string | undefined }) => (
  <span className={cn("size-2 rounded-full", className ?? "border border-muted-foreground")} aria-hidden />
);
