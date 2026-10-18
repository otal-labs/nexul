import { CheckIcon, PlusIcon } from "lucide-react";
import type { ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { PopoverContent } from "@/components/ui/popover";
import { MenuSeparator, menuItemClass } from "@/components/MenuItem";
import { UnreadBadge } from "@/components/UnreadBadge";

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
      // Centres the plus over the 28px tiles above, so the label starts where their names do.
      <div className="flex flex-col [&_button>svg:first-child]:mx-1.5">
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
}

export const SwitcherMenuItem = ({ tile, mark, name, selected, onSelect, unreadCount = 0 }: SwitcherMenuItemProps) => (
  <button type="button" onClick={onSelect} aria-current={selected || undefined} title={name} className={menuItemClass}>
    {mark}
    {!mark && (
      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-accent text-xs font-semibold text-primary">
        {tile}
      </span>
    )}
    <span className="min-w-0 flex-1 truncate">{name}</span>
    <UnreadBadge count={unreadCount} />
    {selected && <CheckIcon className="text-primary" aria-hidden />}
  </button>
);
