import { CheckIcon, PlusIcon } from "lucide-react";
import type { ReactNode } from "react";

import { PopoverContent } from "@/components/ui/popover";
import { MenuSeparator, menuItemClass } from "@/components/MenuItem";
import { UnreadBadge } from "@/components/UnreadBadge";

interface SwitcherMenuProps {
  children: ReactNode;
  createLabel: string;
  // Absent for a viewer who may not create one.
  onCreate?: (() => void) | undefined;
}

export const SwitcherMenu = ({ children, createLabel, onCreate }: SwitcherMenuProps) => (
  <PopoverContent side="bottom" align="start" sideOffset={6} className="w-72 p-1">
    <div className="flex max-h-80 flex-col overflow-y-auto">{children}</div>
    {onCreate && (
      <div className="flex flex-col">
        <MenuSeparator />
        <button type="button" onClick={onCreate} className={menuItemClass}>
          <PlusIcon aria-hidden />
          <span>{createLabel}</span>
        </button>
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
      <span className="flex h-5 min-w-5 shrink-0 items-center justify-center rounded-md bg-accent px-1 text-xs font-semibold text-primary">
        {tile}
      </span>
    )}
    <span className="min-w-0 flex-1 truncate">{name}</span>
    <UnreadBadge count={unreadCount} />
    {selected && <CheckIcon className="text-primary" aria-hidden />}
  </button>
);
