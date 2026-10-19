import { useState, type ReactNode } from "react";
import { Check, Trash2 } from "lucide-react";

import { TargetMark } from "@/components/access/TargetMark";
import type { PanelTarget } from "@/hooks/usePanelTargets";
import { cn } from "@/lib/utils";

interface PanelTargetRowProps {
  target: PanelTarget;
  selected: boolean;
  onSelect: () => void;
  subline: ReactNode;
  onRemove?: () => void;
}

interface RemoveTargetButtonProps {
  name: string;
  shown: boolean;
  onRemove: () => void;
}

// The first press arms it and the second removes; leaving the button disarms it.
const RemoveTargetButton = ({ name, shown, onRemove }: RemoveTargetButtonProps) => {
  const [armed, setArmed] = useState(false);
  return (
    <button
      type="button"
      aria-label={armed ? `Confirm removing ${name}` : `Remove ${name}`}
      title={armed ? "Press again to remove" : "Remove"}
      onClick={() => {
        if (!armed) return setArmed(true);
        onRemove();
      }}
      onBlur={() => setArmed(false)}
      onMouseLeave={() => setArmed(false)}
      className={cn(
        "absolute top-1/2 right-1.5 grid size-6 -translate-y-1/2 place-items-center rounded-md text-muted-foreground transition-[opacity,color,background-color] duration-150 ease-standard hover:bg-destructive/10 hover:text-destructive focus-visible:opacity-100 group-hover:opacity-100 [&_svg]:size-3.5",
        !shown && !armed && "opacity-0",
        armed && "bg-destructive/15 text-destructive",
      )}
    >
      {armed && <Check aria-hidden />}
      {!armed && <Trash2 aria-hidden />}
    </button>
  );
};

// Wide: a list row with the 2px brand edge when selected. At 768px: the same target as a compact tab in a scrolling row.
export const PanelTargetRow = ({ target, selected, onSelect, subline, onRemove }: PanelTargetRowProps) => (
  <li className="group relative shrink-0">
    <button
      type="button"
      aria-current={selected ? "true" : undefined}
      onClick={onSelect}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left transition-colors duration-150 ease-standard @3xl/panel:rounded-none @3xl/panel:border-l-2 @3xl/panel:px-3 @3xl/panel:py-2",
        selected && "bg-accent ring-1 ring-border @3xl/panel:border-l-brand @3xl/panel:ring-0",
        !selected && "hover:bg-accent/40 @3xl/panel:border-l-transparent",
        onRemove && "pr-9 @3xl/panel:pr-9",
      )}
    >
      <TargetMark target={target} />
      <span className="min-w-0 flex-1">
        <span title={target.name} className="block truncate text-[13px] font-medium">
          {target.name}
        </span>
        <span className="hidden truncate font-mono text-[11px] text-muted-foreground @3xl/panel:block">{subline}</span>
      </span>
    </button>
    {onRemove && <RemoveTargetButton name={target.name} shown={selected} onRemove={onRemove} />}
  </li>
);
