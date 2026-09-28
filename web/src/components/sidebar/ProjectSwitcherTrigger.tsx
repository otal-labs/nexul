import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { Project } from "@/models/Project";

interface ProjectSwitcherTriggerProps {
  current: Project;
  collapsed: boolean;
}

export const ProjectSwitcherTrigger = ({ current, collapsed }: ProjectSwitcherTriggerProps) => (
  <PopoverTrigger asChild>
    <button
      type="button"
      title={collapsed ? current.name : undefined}
      className={cn(
        "flex min-w-0 flex-1 items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed && "justify-center px-0",
      )}
    >
      <span className="w-8 shrink-0 text-center font-mono text-[11px] font-semibold text-muted-foreground">
        {current.prefix || current.name[0]?.toUpperCase()}
      </span>
      {!collapsed && (
        <>
          <span className="flex-1 truncate font-medium">{current.name}</span>
          <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </>
      )}
    </button>
  </PopoverTrigger>
);
