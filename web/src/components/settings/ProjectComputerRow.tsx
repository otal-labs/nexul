import { PlusIcon } from "lucide-react";
import { useState } from "react";

import { AddProjectSection } from "@/components/settings/AddProjectSection";
import { ProjectCheckoutCell } from "@/components/settings/ProjectCheckoutCell";
import { RankedComputerRow, type RowControls } from "@/components/settings/RankedComputerRow";
import { Button } from "@/components/ui/button";
import type { Computer } from "@/models/Pairing";
import type { ProjectCheckout } from "@/models/ProjectCheckout";

interface ProjectComputerRowProps {
  projectId: string;
  computer: Computer;
  rank: number;
  checkout: ProjectCheckout;
  // The project's own order offers dragging and removing; following the person's order offers neither.
  controls: RowControls;
  onPick: (harnessProjectId: string) => void;
}

// One computer a project's runs may land on, ranked, with the checkout they would use there and the way to add one.
export const ProjectComputerRow = ({ projectId, computer, rank, checkout, controls, onPick }: ProjectComputerRowProps) => {
  const [adding, setAdding] = useState(false);
  const missing = checkout.state === "missing";

  return (
    <RankedComputerRow
      id={computer.id}
      name={computer.name}
      rank={rank}
      controls={controls}
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
