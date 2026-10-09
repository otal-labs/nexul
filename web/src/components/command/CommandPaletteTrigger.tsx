import { SearchIcon } from "lucide-react";

import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { navLinkClass } from "@/components/SidebarNav";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";
import { commandPaletteShortcut } from "@/lib/shortcut";
import { cn } from "@/lib/utils";

interface CommandPaletteTriggerProps {
  collapsed: boolean;
}

// The palette's way in for a pointer, and the place its shortcut is learned.
export const CommandPaletteTrigger = ({ collapsed }: CommandPaletteTriggerProps) => {
  const openPalette = useCommandPaletteStore((s) => s.setOpen);
  const shortcut = commandPaletteShortcut();
  return (
    <RailTooltip label={`Search  ${shortcut}`} collapsed={collapsed}>
      <button
        type="button"
        onClick={() => openPalette(true)}
        aria-keyshortcuts={shortcut === "⌘K" ? "Meta+K" : "Control+K"}
        aria-label={collapsed ? "Search" : undefined}
        className={cn(navLinkClass({ isActive: false }), "w-full")}
      >
        <span className="flex w-8 shrink-0 justify-center">
          <SearchIcon className="size-4" aria-hidden />
        </span>
        {!collapsed && <span className="flex-1 text-left">Search</span>}
        {!collapsed && (
          <kbd className="rounded-sm border border-border px-1.5 py-px font-mono text-[11px] text-muted-foreground">{shortcut}</kbd>
        )}
      </button>
    </RailTooltip>
  );
};
