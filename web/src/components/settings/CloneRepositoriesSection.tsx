import { useMemo } from "react";

import { CloneRepositoryButton } from "@/components/settings/CloneRepositoryButton";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import type { HarnessProject } from "@/models/Pairing";

interface CloneRepositoriesSectionProps {
  projectId: string;
  computerId: string;
  projects: HarnessProject[];
  onCloned: (harnessProjectId: string) => void;
}

const folderName = (path: string) => path.split(/[\\/]/).pop() ?? "";

// Offers the project's GitHub repositories that no T3 project on the computer holds, matched by folder name.
export const CloneRepositoriesSection = ({ projectId, computerId, projects, onCloned }: CloneRepositoriesSectionProps) => {
  const { data: repos } = useFetchProjectRepos(projectId);
  const missing = useMemo(
    () => (repos ?? []).filter((r) => r.connector_id === "github" && !projects.some((p) => folderName(p.path) === r.name)),
    [repos, projects],
  );

  if (missing.length === 0) return null;

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        Not on this computer yet. Cloning puts it beside the computer's other T3 projects.
      </p>
      <div className="flex flex-wrap gap-2">
        {missing.map((r) => (
          <CloneRepositoryButton key={r.full_name} computerId={computerId} repository={r.full_name} onCloned={onCloned} />
        ))}
      </div>
    </div>
  );
};
