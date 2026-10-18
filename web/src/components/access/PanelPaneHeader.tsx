import { TargetMark } from "@/components/access/TargetMark";
import type { PanelTarget } from "@/hooks/usePanelTargets";

interface PanelPaneHeaderProps {
  target: PanelTarget;
  title: string;
  line: string;
}

// The pane's head: the target's 32px mark, a title and a muted line.
export const PanelPaneHeader = ({ target, title, line }: PanelPaneHeaderProps) => (
  <header className="flex items-center gap-3">
    <TargetMark target={target} className="size-8" />
    <div className="min-w-0 flex-1">
      <h3 className="truncate text-[15px] font-semibold tracking-tight">{title}</h3>
      <p className="truncate text-xs text-muted-foreground">{line}</p>
    </div>
  </header>
);
