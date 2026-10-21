import { useMemo } from "react";

import { CloneRepositoriesSection } from "@/components/settings/CloneRepositoriesSection";
import { CreateEmptyProject } from "@/components/settings/CreateEmptyProject";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import type { HarnessProject } from "@/models/Pairing";

interface AddProjectSectionProps {
  projectId?: string | undefined;
  computerId: string;
  projects: HarnessProject[];
  onAdded: (harnessProjectId: string) => void;
}

const folderName = (path: string) => path.split(/[\\/]/).pop() ?? "";

// Offers the project's GitHub repositories that no T3 project on the computer holds, matched by folder name, and an
// empty project once nothing is left to clone.
export const AddProjectSection = ({ projectId, computerId, projects, onAdded }: AddProjectSectionProps) => {
  const { data: repos, isPending } = useFetchProjectRepos(projectId);
  const missing = useMemo(
    () => (repos ?? []).filter((r) => r.connector_id === "github" && !projects.some((p) => folderName(p.path) === r.name)).map((r) => r.full_name),
    [repos, projects],
  );
  const loaded = !projectId || !isPending;

  return (
    <div className="space-y-2">
      {missing.length > 0 && <CloneRepositoriesSection computerId={computerId} repositories={missing} onCloned={onAdded} />}
      {loaded && missing.length === 0 && <CreateEmptyProject computerId={computerId} onCreated={onAdded} />}
    </div>
  );
};
