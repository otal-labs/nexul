import { PlusIcon } from "lucide-react";

interface TopologyPaletteProps {
  onAddExternal: () => void;
}

// Service nodes, networks, and hostnames are all derived; the only thing left to draw by hand is something
// Nexul doesn't manage. Below `sm` the button collapses to its icon.
export const TopologyPalette = ({ onAddExternal }: TopologyPaletteProps) => (
  <button
    type="button"
    className="animate-in fade-in-0 slide-in-from-top-1 absolute right-2 top-2 z-10 flex items-center gap-2 rounded-lg border border-border bg-card px-2.5 py-1.5 text-sm text-foreground shadow-card transition-colors duration-150 ease-standard hover:bg-accent hover:text-accent-foreground sm:right-4 sm:top-4"
    aria-label="Add external node"
    onClick={onAddExternal}
  >
    <PlusIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    <span className="hidden sm:inline">External</span>
  </button>
);
