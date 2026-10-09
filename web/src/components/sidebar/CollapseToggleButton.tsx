import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { Logo } from "@/components/Logo";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { Button } from "@/components/ui/button";

interface CollapseToggleButtonProps {
  collapsed: boolean;
  onToggle: () => void;
}

// Collapsed, the rail has room for one tile at the top, so the logo is the way back out: it turns into the expand glyph under the pointer or focus.
export const CollapseToggleButton = ({ collapsed, onToggle }: CollapseToggleButtonProps) => (
  <>
    {collapsed && (
      <RailTooltip label="Expand sidebar" collapsed>
        <button
          type="button"
          onClick={onToggle}
          aria-label="Expand sidebar"
          className="group/expand relative grid size-8 place-items-center rounded-lg outline-none focus-visible:ring-[3px] focus-visible:ring-ring/40"
        >
          <Logo className="nav-swap group-hover/expand:opacity-0 group-focus-visible/expand:opacity-0" />
          <span
            aria-hidden
            className="nav-swap absolute inset-0 grid place-items-center rounded-lg bg-accent text-foreground opacity-0 group-hover/expand:opacity-100 group-focus-visible/expand:opacity-100"
          >
            <PanelLeftOpen className="size-4" />
          </span>
        </button>
      </RailTooltip>
    )}
    {!collapsed && (
      <Button variant="ghost" size="icon" className="size-7 text-muted-foreground hover:text-foreground" onClick={onToggle} aria-label="Collapse sidebar" title="Collapse sidebar">
        <PanelLeftClose className="size-4" aria-hidden />
      </Button>
    )}
  </>
);
