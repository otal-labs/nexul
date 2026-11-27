import { PlusIcon } from "lucide-react";
import { useState } from "react";

import { AddProjectSection } from "@/components/settings/AddProjectSection";
import { EmptyProjectPicker } from "@/components/settings/EmptyProjectPicker";
import { ProjectCheckoutCell } from "@/components/settings/ProjectCheckoutCell";
import { RankedComputerRow, type RowControls } from "@/components/settings/RankedComputerRow";
import { Button } from "@/components/ui/button";
import type { Computer } from "@/models/Pairing";
import type { RepoCheckout } from "@/models/ProjectCheckout";

interface ProjectComputerRowProps {
  projectId: string;
  projectName: string;
  computer: Computer;
  rank: number;
  // One per repository of the project, or one for a project with one or none.
  checkouts: RepoCheckout[];
  // The project's own order offers dragging and removing; following the person's order offers neither.
  controls: RowControls;
  onPick: (repository: string, harnessProjectId: string) => void;
}

// One computer a project's runs may land on, ranked, with the checkout each repository's runs would use there and the
// way to add one.
export const ProjectComputerRow = ({ projectId, projectName, computer, rank, checkouts, controls, onPick }: ProjectComputerRowProps) => {
  const [adding, setAdding] = useState(false);
  const firstMissing = checkouts.find((c) => c.checkout.state === "missing");
  const missing = !!firstMissing;
  const projects = computer.facts?.projects ?? [];
  const several = checkouts.length > 1;
  const pick = (id: string, repository?: string) => {
    onPick((repository ?? firstMissing?.repository ?? "").toLowerCase(), id);
    setAdding(false);
  };

  return (
    <RankedComputerRow
      id={computer.id}
      name={computer.name}
      computer={computer}
      rank={rank}
      controls={controls}
      below={
        missing &&
        adding && (
          <div className="settle-in space-y-2 border-t border-border px-3 py-3">
            <EmptyProjectPicker computerName={computer.name} projects={projects} onPick={pick} />
            <AddProjectSection projectId={projectId} computerId={computer.id} projects={projects} onAdded={pick} />
          </div>
        )
      }
    >
      <div className="flex min-w-0 flex-col items-end gap-1">
        {checkouts.map((c) => (
          <ProjectCheckoutCell
            key={c.repository}
            computerName={computer.name}
            repository={several ? c.repository : undefined}
            checkout={c.checkout}
            projectName={projectName}
            onPick={(id) => onPick(c.repository, id)}
          />
        ))}
      </div>
      {missing && (
        <Button type="button" variant="ghost" size="sm" aria-expanded={adding} onClick={() => setAdding(!adding)}>
          <PlusIcon className="size-4" aria-hidden />
          Add
        </Button>
      )}
    </RankedComputerRow>
  );
};
