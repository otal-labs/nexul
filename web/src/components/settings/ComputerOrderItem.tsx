import { useSortable } from "@dnd-kit/sortable";
import { motion, useReducedMotion } from "motion/react";
import { useMemo } from "react";

import { ComputerItem } from "@/components/settings/ComputerItem";
import { ComputerOrderProvider } from "@/components/settings/ComputerOrderContext";
import { RowGrip } from "@/components/settings/RowGrip";
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

// One computer in Your computers: its rank and a grip that drags it at the row's start, then the row itself (ADR 0165).
export const ComputerOrderItem = ({ computer, index, count, presence, onMove }: ComputerOrderItemProps) => {
  const movable = count > 1;
  const reduced = useReducedMotion() ?? false;
  const name = computer.name || "New computer";
  const sortable = useSortable({
    id: computer.id,
    disabled: !movable,
    transition: reduced ? null : { duration: 200, easing: EASE_OUT },
  });
  const { setNodeRef, transform, transition, isDragging } = sortable;
  const order = useMemo(
    () => ({
      grip: movable && (
        <span className="flex shrink-0 items-center gap-1">
          {/* Centred, 12px mono sits 1px above the 14px name's baseline; the nudge puts the rank on it. */}
          <span aria-hidden className="w-5 translate-y-px font-mono text-xs text-muted-foreground tabular-nums">
            #{index + 1}
          </span>
          <RowGrip name={name} sortable={sortable} />
        </span>
      ),
      moveActions: movable
        ? [
            { label: "Move up", disabled: index === 0, onSelect: () => onMove(index, index - 1) },
            { label: "Move down", disabled: index === count - 1, onSelect: () => onMove(index, index + 1) },
          ]
        : [],
    }),
    [movable, sortable, name, index, count, onMove],
  );

  return (
    <li
      ref={setNodeRef}
      style={{ transform: transform ? `translate3d(0, ${Math.round(transform.y)}px, 0)` : undefined, transition }}
      className={cn("relative", isDragging && "z-10 rounded-md bg-card shadow-elevated")}
    >
      {/* A move from the menu glides so the eye follows the row; one made with the keyboard lands at once. */}
      <motion.div layout={!reduced && !lastInputWasKeyboard() ? "position" : false} layoutDependency={index} transition={{ layout: ROW_GLIDE }}>
        <ComputerOrderProvider value={order}>
          <ComputerItem computer={computer} presence={presence} />
        </ComputerOrderProvider>
      </motion.div>
    </li>
  );
};
