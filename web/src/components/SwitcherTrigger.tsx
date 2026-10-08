import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface SwitcherTriggerProps {
  tile: string;
  name: string;
  collapsed: boolean;
}

export const SwitcherTrigger = ({ tile, name, collapsed }: SwitcherTriggerProps) => (
  <PopoverTrigger asChild>
    <button
      type="button"
      title={name}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md text-left outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed ? "justify-center px-0 py-1" : "px-2.5 py-1.5",
      )}
    >
      <span className="flex h-8 min-w-8 shrink-0 items-center justify-center rounded-lg bg-accent px-1 text-xs font-semibold text-primary">
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
