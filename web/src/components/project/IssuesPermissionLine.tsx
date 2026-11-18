import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useIssuesPermissionMissing } from "@/hooks/IssueSyncHooks";
import type { IssueSync } from "@/models/IssueSync";

interface IssuesPermissionLineProps {
  sync: IssueSync;
}

// Under the repositories, for one whose issues are switched on while the App's installation can't read them yet.
export const IssuesPermissionLine = ({ sync }: IssuesPermissionLineProps) => {
  const missing = useIssuesPermissionMissing(sync.owner);
  if (!missing) return null;
  return (
    <div className="mt-2 px-2">
      <SettingsStatus tone="warning" detail={`${sync.owner}/${sync.repo}`}>
        Issues permission missing
      </SettingsStatus>
      <p className="text-xs text-muted-foreground">
        Give the App Issues: Read and write in its settings on GitHub, then accept the change here under Manage.
      </p>
    </div>
  );
};
