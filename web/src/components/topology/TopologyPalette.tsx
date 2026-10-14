import { PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";

interface TopologyPaletteProps {
  onAddExternal: () => void;
}

// Service nodes, networks, and hostnames are all derived; the only thing left to draw by hand is something
// Nexul doesn't manage. Below `sm` the button collapses to its icon.
export const TopologyPalette = ({ onAddExternal }: TopologyPaletteProps) => (
  <ActionButton
    permission="topology:write"
    verb="add topology nodes"
    variant="ghost"
    icon={PlusIcon}
    className="animate-in fade-in-0 slide-in-from-top-1 canvas-card absolute right-2 top-2 z-10 h-auto gap-2 px-2.5 py-1.5 font-normal text-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-accent-foreground sm:right-4 sm:top-4"
    aria-label="Add external node"
    onClick={onAddExternal}
  >
    <span className="hidden sm:inline">External</span>
  </ActionButton>
);
