import type { ReactNode } from "react";

import { TargetMark } from "@/components/access/TargetMark";
import type { PanelTarget } from "@/hooks/usePanelTargets";

interface PanelPaneHeaderProps {
  target: PanelTarget;
  title: string;
  line: string;
  action?: ReactNode;
}

// The pane's head: the target's 32px mark, a title and a muted line, an optional action on the right.
export const PanelPaneHeader = ({ target, title, line, action }: PanelPaneHeaderProps) => (
  <header className="flex items-center gap-3">
    <TargetMark target={target} className="size-8" />
    <div className="min-w-0 flex-1">
      <h3 className="truncate text-[15px] font-semibold tracking-tight">{title}</h3>
      <p className="truncate text-xs text-muted-foreground">{line}</p>
    </div>
    {action}
  </header>
);
