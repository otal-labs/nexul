import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { Button } from "@/components/ui/button";

interface CollapseToggleButtonProps {
  collapsed: boolean;
  onToggle: () => void;
}

export const CollapseToggleButton = ({ collapsed, onToggle }: CollapseToggleButtonProps) => (
  <Button
    variant="ghost"
    size="icon"
    className="size-7 text-muted-foreground hover:text-foreground"
    onClick={onToggle}
    aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
  >
    {collapsed && <PanelLeftOpen className="size-4" aria-hidden />}
    {!collapsed && <PanelLeftClose className="size-4" aria-hidden />}
  </Button>
);
