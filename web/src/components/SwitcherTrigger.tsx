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
      title={collapsed ? name : undefined}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed && "justify-center",
      )}
    >
      <span className="flex h-8 min-w-8 shrink-0 items-center justify-center rounded-lg bg-accent px-1 text-[12px] font-semibold text-primary">
        {tile}
      </span>
      {!collapsed && (
        <>
          <span className="flex-1 truncate text-[13px] font-medium">{name}</span>
          <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </>
      )}
    </button>
  </PopoverTrigger>
);
