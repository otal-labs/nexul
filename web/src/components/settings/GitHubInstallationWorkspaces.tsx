import { X } from "lucide-react";

import { GitHubInstallationAssign } from "@/components/settings/GitHubInstallationAssign";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useUnassignInstallation } from "@/hooks/RepositoryHooks";
import type { Installation } from "@/models/Repository";

interface GitHubInstallationWorkspacesProps {
  installation: Installation;
  canManage: boolean;
}

// The workspaces that list an installation's repositories; a connector manager adds one or removes one with its ×.
export const GitHubInstallationWorkspaces = ({ installation, canManage }: GitHubInstallationWorkspacesProps) => {
  const unassign = useUnassignInstallation();

  return (
    <div className="mt-1 flex flex-wrap items-center gap-1.5">
      {installation.workspaces.length === 0 && !installation.gone && (
        <SettingsStatus tone="warning" detail="no workspace lists its repositories">
          Unassigned
        </SettingsStatus>
      )}
      {installation.workspaces.map((ws) => (
        <span key={ws.id} className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs">
          {ws.name}
          {canManage && (
            <button
              type="button"
              aria-label={`Stop ${ws.name} listing ${installation.account_login}'s repositories`}
              className="rounded-full text-muted-foreground hover:text-foreground"
              disabled={unassign.isPending}
              onClick={() => unassign.mutate({ account: installation.account_login, workspaceId: ws.id })}
            >
              <X className="size-3" aria-hidden />
            </button>
          )}
        </span>
      ))}
      {canManage && !installation.gone && <GitHubInstallationAssign installation={installation} />}
    </div>
  );
};
