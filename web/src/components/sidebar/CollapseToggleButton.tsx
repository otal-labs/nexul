import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { Button } from "@/components/ui/button";

interface CollapseToggleButtonProps {
  collapsed: boolean;
  onToggle: () => void;
}

export const CollapseToggleButton = ({ collapsed, onToggle }: CollapseToggleButtonProps) => (
  <Button
    variant="outline"
    size="icon"
    className="absolute -right-3 top-4 z-10 size-6 rounded-full bg-surface-2 shadow-sm"
    onClick={onToggle}
    aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
  >
    {collapsed ? (
      <PanelLeftOpen className="size-3.5" aria-hidden />
    ) : (
      <PanelLeftClose className="size-3.5" aria-hidden />
    )}
  </Button>
);
