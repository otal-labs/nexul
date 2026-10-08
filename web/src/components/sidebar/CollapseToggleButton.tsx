import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface CollapseToggleButtonProps {
  collapsed: boolean;
  onToggle: () => void;
}

// Collapsed, the 56px rail has no room beside the logo, so the toggle straddles its right border on the logo's line.
export const CollapseToggleButton = ({ collapsed, onToggle }: CollapseToggleButtonProps) => (
  <Button
    variant={collapsed ? "outline" : "ghost"}
    size="icon"
    className={cn(
      "size-7 text-muted-foreground hover:text-foreground",
      collapsed && "absolute top-1/2 -right-3 size-6 -translate-y-1/2 rounded-full bg-popover shadow-sm",
    )}
    onClick={onToggle}
    aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
  >
    {collapsed && <PanelLeftOpen className="size-3.5" aria-hidden />}
    {!collapsed && <PanelLeftClose className="size-4" aria-hidden />}
  </Button>
);
