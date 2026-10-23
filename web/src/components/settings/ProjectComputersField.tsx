import { useFormContext, useWatch } from "react-hook-form";

import { ProjectComputerRow } from "@/components/settings/ProjectComputerRow";
import { RankedComputersField } from "@/components/settings/RankedComputersField";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import type { ProjectLinkFormData } from "@/models/Pairing";
import { projectCheckout, projectRemotes, withCheckout } from "@/models/ProjectCheckout";

interface ProjectComputersFieldProps {
  projectId: string;
}

// The computers a project's runs try, ranked: the person's order, or the project's own list in its own order, each with
// the checkout a run would use there (ADR 0165).
export const ProjectComputersField = ({ projectId }: ProjectComputersFieldProps) => {
  const { control, setValue } = useFormContext<ProjectLinkFormData>();
  const ids = useWatch({ control, name: "computers" });
  const checkouts = useWatch({ control, name: "checkouts" });
  const { data: own = [] } = useListComputers();
  const { data: repos = [] } = useFetchProjectRepos(projectId);
  const remotes = projectRemotes(repos);
  const dirty = { shouldDirty: true };

  return (
    <div className="pb-2">
      <RankedComputersField
        subject="project"
        label="Computers"
        own={own}
        ids={ids}
        onChange={(next) => setValue("computers", next, dirty)}
        renderRow={({ id, rank, controls }) => {
          const computer = own.find((c) => c.id === id);
          return (
            computer && (
              <ProjectComputerRow
                projectId={projectId}
                computer={computer}
                rank={rank}
                checkout={projectCheckout(computer, checkouts, remotes)}
                controls={controls}
                onPick={(harnessProjectId) => setValue("checkouts", withCheckout(checkouts, computer.id, harnessProjectId), dirty)}
              />
            )
          );
        }}
      />
    </div>
  );
};
