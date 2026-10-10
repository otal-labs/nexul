import { X } from "lucide-react";

import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useDetachInstallation } from "@/hooks/RepositoryHooks";
import type { InstallationWorkspace } from "@/models/Repository";

interface GitHubInstallationWorkspaceChipProps {
  account: string;
  workspace: InstallationWorkspace;
}

// One workspace using an account; × detaches it there, after a confirm, where the viewer manages projects.
export const GitHubInstallationWorkspaceChip = ({ account, workspace }: GitHubInstallationWorkspaceChipProps) => {
  const detach = useDetachInstallation();
  const { open: confirm } = useConfirmationDialog();

  const onDetach = async () => {
    const ok = await confirm({
      title: `Detach ${account} from ${workspace.name}?`,
      message: `Deploys, webhooks and pull request reads in ${workspace.name} stop reading ${account}'s repositories until someone who can open one attaches it to a project again.`,
      confirmLabel: "Detach",
    });
    if (ok) detach.mutate({ account, workspaceId: workspace.id });
  };

  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs">
      <span className="text-muted-foreground">Used by</span>
      {workspace.name}
      {workspace.can_detach && (
        <button
          type="button"
          aria-label={`Detach ${account} from ${workspace.name}`}
          title={`Detach ${account} from ${workspace.name}`}
          className="rounded-full text-muted-foreground hover:text-foreground"
          disabled={detach.isPending}
          onClick={() => void onDetach()}
        >
          <X className="size-3" aria-hidden />
        </button>
      )}
    </span>
  );
};
