import { Rocket } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { GithubMark } from "@/components/ProviderMarks";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { wizardDoorPermissions } from "@/models/Access";
import type { RepoRef } from "@/models/Project";

interface WaitingRepositoryRowProps {
  projectId: string;
  repo: RepoRef;
}

// An attached repository no service builds from yet, with the way into the wizard's Service step for it.
export const WaitingRepositoryRow = ({ projectId, repo }: WaitingRepositoryRowProps) => {
  const wsPath = useWorkspacePath();
  const to = wsPath(`/wizard/project/repository?project=${projectId}&add=1&repo=${encodeURIComponent(repo.full_name)}`);
  return (
    <SettingsListRow
      lead={<GithubMark className="size-3.5 text-muted-foreground" />}
      name={<span className="truncate font-mono text-[13px]" title={repo.full_name}>{repo.full_name}</span>}
      meta="Attached, not deployed yet"
      showActions
      actions={
        <ActionButton permission={wizardDoorPermissions(false)} projectId={projectId} verb="add a service" variant="outline" size="sm" icon={Rocket} to={to}>
          Deploy
        </ActionButton>
      }
    />
  );
};
