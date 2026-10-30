import { useState } from "react";

import { GithubMark } from "@/components/ProviderMarks";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { useRemoveProjectRepo } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { RepoRole } from "@/enums/Project";
import type { RepoRef } from "@/models/Project";
import { countLabel } from "@/utils/CountUtility";

interface RepoRowProps {
  projectId: string;
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
  repo: RepoRef;
  // How many of the project's services build from it; unknown until they load, or for a viewer without stacks.
  services: number | undefined;
}

export const RepoRow = ({ projectId, onLeave, repo, services }: RepoRowProps) => {
  const removeRepo = useRemoveProjectRepo();
  const [leaving, setLeaving] = useState(false);
  const { open: confirm } = useConfirmationDialog();

  const remove = async () => {
    const ok = await confirm({
      title: `Remove ${repo.full_name}?`,
      message: "Its pull requests stop linking to this project's tickets. The repository itself isn't touched, and you can add it again.",
      confirmLabel: "Remove repo",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave?.();
    removeRepo.mutate({ owner: repo.owner, name: repo.name }, { onError: () => setLeaving(false) });
  };

  const tests = repo.role === RepoRole.Tests;
  return (
    <SettingsListRow
      data-leaving={leaving || undefined}
      className={leavingRowClass}
      lead={<GithubMark className="size-3.5 text-muted-foreground" />}
      name={<span className="truncate font-mono text-[13px]" title={repo.full_name}>{repo.full_name}</span>}
      meta={tests ? "Tests, never deployed" : "App"}
      count={services !== undefined && countLabel(services, "service")}
      actions={
        <RowActionsMenu
          subject={repo.full_name}
          projectId={projectId}
          actions={[{ label: "Remove", destructive: true, permission: "projects:write", verb: "remove repositories", onSelect: () => void remove() }]}
        />
      }
    />
  );
};
