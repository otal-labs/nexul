import { GitHubInstallationWorkspaceChip } from "@/components/settings/GitHubInstallationWorkspaceChip";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { Installation } from "@/models/Repository";

interface GitHubInstallationWorkspacesProps {
  installation: Installation;
}

// The workspaces whose projects attach one of the account's repositories.
export const GitHubInstallationWorkspaces = ({ installation }: GitHubInstallationWorkspacesProps) => (
  <div className="mt-1 flex flex-wrap items-center gap-1.5">
    {installation.workspaces.length === 0 && <SettingsStatus tone="muted">Not used by a workspace yet</SettingsStatus>}
    {installation.workspaces.map((ws) => (
      <GitHubInstallationWorkspaceChip key={ws.id} account={installation.account_login} workspace={ws} />
    ))}
  </div>
);
