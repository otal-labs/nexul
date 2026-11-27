import { CheckIcon, LoaderCircle, PlusIcon } from "lucide-react";
import type { ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { PopoverContent } from "@/components/ui/popover";
import { MenuSeparator, menuItemClass } from "@/components/MenuItem";
import { UnreadBadge } from "@/components/UnreadBadge";
import { cn } from "@/lib/utils";

// At the sidebar row's metrics, so each name starts where the trigger's does: 28px tall, 13px, the mark centred in a 32px column.
const switcherRowClass = cn(
  menuItemClass,
  "min-h-7 px-1.5 py-0.5 text-row text-muted-foreground aria-[current=true]:font-medium aria-[current=true]:text-foreground",
);

interface SwitcherMenuProps {
  children: ReactNode;
  createLabel: string;
  // Absent for a viewer who may not create one.
  onCreate?: (() => void) | undefined;
  // What creating needs; given, the row is locked or hidden by it instead of by the caller leaving onCreate out.
  createGate?: ActionGate | undefined;
  // On the collapsed rail it opens beside the rail rather than down over the icons under it.
  collapsed: boolean;
}

export const SwitcherMenu = ({ children, createLabel, onCreate, createGate, collapsed }: SwitcherMenuProps) => (
  <PopoverContent
    side={collapsed ? "right" : "bottom"}
    align="start"
    sideOffset={collapsed ? 8 : 6}
    className="w-[max(var(--radix-popover-trigger-width),14rem)] p-1"
  >
    <div className="flex max-h-80 flex-col overflow-y-auto">{children}</div>
    {onCreate && (
      // The create row takes the switcher row's metrics too, its plus centred in the marks' column.
      <div className="flex flex-col [&_button]:min-h-7 [&_button]:px-1.5 [&_button]:py-0.5 [&_button]:text-row [&_button]:text-muted-foreground [&_button>svg:first-child]:mx-2">
        {createGate && (
          <ActionMenuItem {...createGate} icon={PlusIcon} separated onSelect={onCreate}>
            {createLabel}
          </ActionMenuItem>
        )}
        {!createGate && <MenuSeparator />}
        {!createGate && (
          <button type="button" onClick={onCreate} className={menuItemClass}>
            <PlusIcon aria-hidden />
            <span>{createLabel}</span>
          </button>
        )}
      </div>
    )}
  </PopoverContent>
);

interface SwitcherMenuItemProps {
  tile: string;
  // Stands in for the initial tile: a project shows its mark.
  mark?: ReactNode;
  name: string;
  selected: boolean;
  onSelect: () => void;
  unreadCount?: number;
  runningCount?: number;
}

export const SwitcherMenuItem = ({ tile, mark, name, selected, onSelect, unreadCount = 0, runningCount = 0 }: SwitcherMenuItemProps) => (
  <button type="button" onClick={onSelect} aria-current={selected || undefined} title={name} className={switcherRowClass}>
    <span className="flex w-8 shrink-0 justify-center">
      {mark}
      {!mark && (
        <span className="flex size-6 items-center justify-center rounded-md bg-accent text-micro font-semibold text-primary">{tile}</span>
      )}
    </span>
    <span className="min-w-0 flex-1 truncate">{name}</span>
    {runningCount > 0 && (
      <span
        role="img"
        aria-label={`${runningCount} ${runningCount === 1 ? "play" : "plays"} running`}
        className="flex shrink-0 items-center gap-1 font-mono text-xs leading-none text-warning tabular-nums"
      >
        <LoaderCircle className="size-3 animate-spin text-warning motion-reduce:animate-none" aria-hidden />
        {runningCount}
      </span>
    )}
    <UnreadBadge count={unreadCount} />
    {selected && <CheckIcon className="text-primary" aria-hidden />}
  </button>
);
