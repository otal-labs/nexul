import { CheckIcon, PlusIcon } from "lucide-react";
import type { ReactNode } from "react";

import { PopoverContent } from "@/components/ui/popover";

const menuItemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60 focus-visible:text-foreground";

interface SwitcherMenuProps {
  children: ReactNode;
  createLabel: string;
  // Absent for a viewer who may not create one.
  onCreate?: (() => void) | undefined;
}

export const SwitcherMenu = ({ children, createLabel, onCreate }: SwitcherMenuProps) => (
  <PopoverContent side="bottom" align="start" sideOffset={6} className="w-56 p-1.5">
    <div className="flex max-h-80 flex-col gap-0.5 overflow-y-auto">{children}</div>
    {onCreate && (
      <div className="mt-0.5 flex flex-col gap-0.5">
        <div className="my-1 border-t border-border" />
        <button type="button" onClick={onCreate} className={menuItemClass}>
          <PlusIcon className="size-4 shrink-0" aria-hidden />
          <span>{createLabel}</span>
        </button>
      </div>
    )}
  </PopoverContent>
);

interface SwitcherMenuItemProps {
  tile: string;
  name: string;
  selected: boolean;
  onSelect: () => void;
}

export const SwitcherMenuItem = ({ tile, name, selected, onSelect }: SwitcherMenuItemProps) => (
  <button type="button" onClick={onSelect} aria-current={selected || undefined} className={menuItemClass}>
    <span className="flex h-5 min-w-5 shrink-0 items-center justify-center rounded bg-accent px-1 text-[10px] font-semibold text-primary">
      {tile}
    </span>
    <span className="flex-1 truncate">{name}</span>
    {selected && <CheckIcon className="size-3.5 shrink-0 text-primary" aria-hidden />}
  </button>
);
