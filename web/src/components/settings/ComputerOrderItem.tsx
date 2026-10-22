import { useSortable } from "@dnd-kit/sortable";
import { GripVertical } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
import { useMemo } from "react";

import { ComputerItem } from "@/components/settings/ComputerItem";
import { ComputerOrderProvider } from "@/components/settings/ComputerOrderContext";
import { EASE_OUT, lastInputWasKeyboard, ROW_GLIDE } from "@/lib/motion";
import { cn } from "@/lib/utils";
import type { Computer } from "@/models/Pairing";

interface ComputerOrderItemProps {
  computer: Computer;
  index: number;
  count: number;
  presence?: string | undefined;
  onMove: (from: number, to: number) => void;
}

// One computer in Your computers: its rank in the gutter, a grip that drags it, and the row itself (ADR 0165).
export const ComputerOrderItem = ({ computer, index, count, presence, onMove }: ComputerOrderItemProps) => {
  const movable = count > 1;
  const reduced = useReducedMotion() ?? false;
  const name = computer.name || "New computer";
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: computer.id,
    disabled: !movable,
    transition: reduced ? null : { duration: 200, easing: EASE_OUT },
  });
  const order = useMemo(
    () => ({
      grip: movable && (
        <button
          type="button"
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label={`Reorder ${name}`}
          className="-ml-1 flex size-6 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted-foreground/70 transition-colors duration-150 ease-standard hover:text-foreground focus-visible:outline-2 focus-visible:outline-focus active:cursor-grabbing"
        >
          <GripVertical className="size-4" aria-hidden />
        </button>
      ),
      moveActions: movable
        ? [
            { label: "Move up", disabled: index === 0, onSelect: () => onMove(index, index - 1) },
            { label: "Move down", disabled: index === count - 1, onSelect: () => onMove(index, index + 1) },
          ]
        : [],
    }),
    [movable, setActivatorNodeRef, attributes, listeners, name, index, count, onMove],
  );

  return (
    <li
      ref={setNodeRef}
      style={{ transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined, transition }}
      className={cn("relative bg-card first:rounded-t-md last:rounded-b-md", isDragging && "z-10 rounded-md shadow-elevated")}
    >
      <span aria-hidden className="absolute top-3.5 -left-10 w-8 text-right font-mono text-xs text-muted-foreground tabular-nums">
        #{index + 1}
      </span>
      {/* A move from the menu glides so the eye follows the row; one made with the keyboard lands at once. */}
      <motion.div layout={!reduced && !lastInputWasKeyboard() ? "position" : false} layoutDependency={index} transition={{ layout: ROW_GLIDE }}>
        <ComputerOrderProvider value={order}>
          <ComputerItem computer={computer} presence={presence} />
        </ComputerOrderProvider>
      </motion.div>
    </li>
  );
};
