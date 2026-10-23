import { useSortable } from "@dnd-kit/sortable";
import { useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

import { RemoveIcon } from "@/components/autoplay/ComposerControls";
import { RowGrip } from "@/components/settings/RowGrip";
import { EASE_OUT } from "@/lib/motion";
import { cn } from "@/lib/utils";

// What a ranked list lets a row do: be dragged to another place, and be taken out.
export interface RowControls {
  sortable: boolean;
  onRemove?: (() => void) | undefined;
}

interface RankedComputerRowProps {
  id: string;
  name: string;
  rank: number;
  controls: RowControls;
  // The row's own cells, between the name and its remove button.
  children?: ReactNode;
  // Folds open under the row.
  below?: ReactNode;
}

// One computer in a ranked list: a grip to drag it, its place, its name, and the button that takes it out.
export const RankedComputerRow = ({ id, name, rank, controls, children, below }: RankedComputerRowProps) => {
  const reduced = useReducedMotion() ?? false;
  const sortable = useSortable({ id, disabled: !controls.sortable, transition: reduced ? null : { duration: 200, easing: EASE_OUT } });
  const { setNodeRef, transform, transition, isDragging } = sortable;

  return (
    <li
      ref={setNodeRef}
      style={{ transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined, transition }}
      className={cn("relative bg-card first:rounded-t-md last:rounded-b-md", isDragging && "z-10 rounded-md shadow-elevated")}
    >
      <div className="flex min-h-10 items-center gap-3 px-3 py-1.5">
        {controls.sortable && <RowGrip name={name} sortable={sortable} />}
        <span aria-hidden className="w-4 shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
          {rank}
        </span>
        <p className="min-w-0 flex-1 truncate text-sm" title={name}>
          {name}
        </p>
        {children}
        {controls.onRemove && <RemoveIcon label={`Remove ${name}`} onClick={controls.onRemove} />}
      </div>
      {below}
    </li>
  );
};
