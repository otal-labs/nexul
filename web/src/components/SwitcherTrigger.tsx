import type { ReactNode } from "react";
import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import { RailTooltip } from "@/components/sidebar/RailTooltip";

interface SwitcherTriggerProps {
  tile: string;
  name: string;
  collapsed: boolean;
  // Stands in for the initial tile: a project shows its mark.
  mark?: ReactNode;
}

export const SwitcherTrigger = ({ tile, name, collapsed, mark }: SwitcherTriggerProps) => (
  <RailTooltip label={name} collapsed={collapsed}>
    <PopoverTrigger asChild>
      <button
        type="button"
        title={collapsed ? undefined : name}
        aria-label={collapsed ? name : undefined}
        className="nav-row flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left hover:bg-accent/60 active:bg-accent data-[state=open]:bg-accent"
      >
        {mark}
        {!mark && (
          <span className="flex h-8 min-w-8 shrink-0 items-center justify-center rounded-lg bg-accent px-1 text-xs font-semibold text-primary">
            {tile}
          </span>
        )}
        {!collapsed && (
          <>
            <span className="min-w-0 flex-1 truncate text-sm font-medium">{name}</span>
            <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </>
        )}
      </button>
    </PopoverTrigger>
  </RailTooltip>
);
