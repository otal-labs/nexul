import { IssuesPermissionMissing } from "@/components/settings/IssuesPermissionMissing";
import { useIssuesPermissionMissing } from "@/hooks/IssueSyncHooks";
import type { RepoRef } from "@/models/Project";

interface ReposIssuesNoticeProps {
  repos: readonly RepoRef[];
}

// Why a repository's Issues switch stays off: its account's App installation has not accepted the permission.
export const ReposIssuesNotice = ({ repos }: ReposIssuesNoticeProps) => {
  const missing = useIssuesPermissionMissing(repos.filter((r) => r.connector_id === "github").map((r) => r.owner));
  return missing.length > 0 && <IssuesPermissionMissing accept="under Manage in Settings, Connectors, Installations" />;
};
