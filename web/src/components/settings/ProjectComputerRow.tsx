import { PlusIcon } from "lucide-react";
import { useState } from "react";

import { AddProjectSection } from "@/components/settings/AddProjectSection";
import { ProjectCheckoutCell } from "@/components/settings/ProjectCheckoutCell";
import { RankedComputerRow } from "@/components/settings/RankedComputerRow";
import type { RowAction } from "@/components/settings/RowActionsMenu";
import { Button } from "@/components/ui/button";
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

  return (
    <RankedComputerRow
      name={computer.name}
      rank={rank}
      actions={actions}
      below={
        missing &&
        adding && (
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
        )
      }
    >
      <ProjectCheckoutCell computerName={computer.name} checkout={checkout} onPick={onPick} />
      {missing && (
        <Button type="button" variant="ghost" size="sm" aria-expanded={adding} onClick={() => setAdding(!adding)}>
          <PlusIcon className="size-4" aria-hidden />
          Add
        </Button>
      )}
    </RankedComputerRow>
  );
};
