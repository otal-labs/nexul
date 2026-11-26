import { Switch } from "@/components/ui/switch";
import { useActionState } from "@/hooks/useActionState";
import { useFetchIssueSyncs, useIssuesPermissionMissing, useSetIssueSync } from "@/hooks/IssueSyncHooks";
import { issueSyncOf } from "@/models/IssueSync";
import type { RepoRef } from "@/models/Project";

interface RepoIssuesSwitchProps {
  projectId: string;
  repo: RepoRef;
}

// On, the repository's issues labelled nexul become this project's tickets and follow GitHub; off, they stop.
export const RepoIssuesSwitch = ({ projectId, repo }: RepoIssuesSwitchProps) => {
  const { data: syncs } = useFetchIssueSyncs(projectId);
  const setSync = useSetIssueSync(projectId);
  const action = useActionState({ permission: "projects:write", projectId, verb: "sync issues" });
  const missing = useIssuesPermissionMissing([repo.owner]).length > 0;
  const on = issueSyncOf(syncs, repo.owner, repo.name)?.enabled ?? false;
  return (
    <label className="flex items-center gap-2 text-xs text-muted-foreground">
      Issues
      <Switch
        aria-label={`Sync ${repo.full_name}'s GitHub issues to tickets`}
        checked={on}
        disabled={action.state !== "live" || !syncs || (missing && !on)}
        onCheckedChange={(enabled) => setSync.mutate({ owner: repo.owner, repo: repo.name, enabled })}
      />
    </label>
  );
};
