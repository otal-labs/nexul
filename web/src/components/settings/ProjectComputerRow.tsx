import { PlusIcon } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
import { useState } from "react";

import { AddProjectSection } from "@/components/settings/AddProjectSection";
import { ProjectCheckoutCell } from "@/components/settings/ProjectCheckoutCell";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { Button } from "@/components/ui/button";
import { lastInputWasKeyboard, ROW_GLIDE } from "@/lib/motion";
import type { Computer } from "@/models/Pairing";
import type { ProjectCheckout } from "@/models/ProjectCheckout";

interface ProjectComputerRowProps {
  projectId: string;
  computer: Computer;
  rank: number;
  checkout: ProjectCheckout;
  // The project's own order offers moving and removing; following the person's order offers neither.
  actions: RowAction[];
  onPick: (harnessProjectId: string) => void;
}

// One computer a project's runs may land on, ranked, with the checkout they would use there and the way to add one.
export const ProjectComputerRow = ({ projectId, computer, rank, checkout, actions, onPick }: ProjectComputerRowProps) => {
  const [adding, setAdding] = useState(false);
  const missing = checkout.state === "missing";
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
        <p className="min-w-0 flex-1 truncate text-sm" title={computer.name}>
          {computer.name}
        </p>
        <ProjectCheckoutCell computerName={computer.name} checkout={checkout} onPick={onPick} />
        {missing && (
          <Button type="button" variant="ghost" size="sm" aria-expanded={adding} onClick={() => setAdding(!adding)}>
            <PlusIcon className="size-4" aria-hidden />
            Add
          </Button>
        )}
        {actions.length > 0 && <RowActionsMenu subject={computer.name} actions={actions} />}
      </div>
      {missing && adding && (
        <div className="settle-in border-t border-border px-3 py-3">
          <AddProjectSection
            projectId={projectId}
            computerId={computer.id}
            projects={computer.facts?.projects ?? []}
            onAdded={(id) => {
              onPick(id);
              setAdding(false);
            }}
          />
        </div>
      )}
    </motion.li>
  );
};
