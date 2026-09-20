import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { Workspace } from "@/models/Workspace";

interface WorkspaceSwitcherTriggerProps {
  current: Workspace;
  collapsed: boolean;
}

export const WorkspaceSwitcherTrigger = ({ current, collapsed }: WorkspaceSwitcherTriggerProps) => (
  <PopoverTrigger asChild>
    <button
      type="button"
      title={collapsed ? current.name : undefined}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed && "justify-center",
      )}
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-accent text-[12px] font-semibold text-primary">
        {current.name[0]}
      </span>
      {!collapsed && (
        <>
          <span className="flex-1 truncate text-[13px] font-medium">{current.name}</span>
          <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </>
      )}
    </button>
  </PopoverTrigger>
);
