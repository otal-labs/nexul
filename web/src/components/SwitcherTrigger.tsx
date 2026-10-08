import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface SwitcherTriggerProps {
  tile: string;
  name: string;
  collapsed: boolean;
  // A project's tile is its ticket prefix, technical data, so it sets in mono a step smaller than the workspace's.
  prefix?: boolean;
}

export const SwitcherTrigger = ({ tile, name, collapsed, prefix = false }: SwitcherTriggerProps) => (
  <PopoverTrigger asChild>
    <button
      type="button"
      title={name}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md text-left outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed ? "justify-center px-0 py-1" : "px-2.5 py-1.5",
      )}
    >
      <span
        className={cn(
          "flex shrink-0 items-center justify-center bg-accent px-1 text-primary",
          prefix ? "h-7 min-w-8 rounded-md font-mono text-[11px] font-medium" : "h-8 min-w-8 rounded-lg text-xs font-semibold",
        )}
      >
        {tile}
      </span>
      {!collapsed && (
        <>
          <span className="min-w-0 flex-1 truncate text-sm font-medium">{name}</span>
          <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </>
      )}
    </button>
  </PopoverTrigger>
);
