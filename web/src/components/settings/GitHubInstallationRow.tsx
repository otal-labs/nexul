import { ExternalLink } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { GitHubInstallationWorkspaces } from "@/components/settings/GitHubInstallationWorkspaces";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import type { Installation } from "@/models/Repository";

interface GitHubInstallationRowProps {
  installation: Installation;
}

const repositoryLabel = (installation: Installation): string => {
  if (installation.repository_selection === "all") return "All repositories";
  const count = installation.repository_count;
  if (count === undefined) return "Selected repositories";
  return `${count} selected ${count === 1 ? "repository" : "repositories"}`;
};

export const GitHubInstallationRow = ({ installation }: GitHubInstallationRowProps) => {
  return (
    <li className="flex items-center gap-3 px-3 py-2.5">
      <PersonAvatar login={installation.account_login} src={installation.account_avatar_url} className="size-6" />
      <div className="min-w-0 flex-1">
        <p className="truncate font-mono text-sm">{installation.account_login}</p>
        <p className="truncate text-xs text-muted-foreground">
          {installation.account_type === "organization" ? "Organisation" : "User"}
          {" · "}
          <span className="tabular-nums">{repositoryLabel(installation)}</span>
        </p>
        {installation.problem && (
          <SettingsStatus tone="warning" detail={installation.problem} className="flex max-w-full">
            Not readable
          </SettingsStatus>
        )}
        <GitHubInstallationWorkspaces installation={installation} />
      </div>
      <Button asChild variant="ghost" size="sm">
        <a
          href={installation.html_url}
          target="_blank"
          rel="noreferrer"
          aria-label={`Manage ${installation.account_login} on GitHub`}
        >
          Manage
          <ExternalLink className="size-3.5" aria-hidden />
        </a>
      </Button>
    </li>
  );
};
