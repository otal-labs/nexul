import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { lastInputWasKeyboard, ROW_GLIDE } from "@/lib/motion";

interface RankedComputerRowProps {
  name: string;
  rank: number;
  // The row's own cells, between the name and its menu.
  children?: ReactNode;
  // Offers moving and removing; none leaves the order as it is, with no menu.
  actions: RowAction[];
  // Folds open under the row.
  below?: ReactNode;
}

// One computer in a ranked list: its place, its name, and the menu that reorders it.
export const RankedComputerRow = ({ name, rank, children, actions, below }: RankedComputerRowProps) => {
  const reduced = useReducedMotion() ?? false;

  return (
    <motion.li
      layout={!reduced && !lastInputWasKeyboard() ? "position" : false}
      layoutDependency={rank}
      transition={{ layout: ROW_GLIDE }}
      className="first:rounded-t-md last:rounded-b-md"
    >
      <div className="flex min-h-10 items-center gap-3 px-3 py-1.5">
        <span aria-hidden className="w-4 shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
          {rank}
        </span>
        <p className="min-w-0 flex-1 truncate text-sm" title={name}>
          {name}
        </p>
        {children}
        {actions.length > 0 && <RowActionsMenu subject={name} actions={actions} />}
      </div>
      {below}
    </motion.li>
  );
};
