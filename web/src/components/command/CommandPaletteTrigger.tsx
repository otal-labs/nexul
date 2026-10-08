import { SearchIcon } from "lucide-react";

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
    <button
      type="button"
      onClick={() => openPalette(true)}
      aria-keyshortcuts={shortcut === "⌘K" ? "Meta+K" : "Control+K"}
      title={collapsed ? `Search (${shortcut})` : undefined}
      aria-label={collapsed ? "Search" : undefined}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md text-sm text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed ? "justify-center py-1.5" : "px-2.5 py-1.5",
      )}
    >
      <span className="flex w-8 shrink-0 justify-center">
        <SearchIcon className="size-4" aria-hidden />
      </span>
      {!collapsed && <span className="flex-1 text-left">Search</span>}
      {!collapsed && (
        <kbd className="rounded-sm border border-border px-1.5 py-px font-mono text-[11px] text-muted-foreground">{shortcut}</kbd>
      )}
    </button>
  );
};
