import { ActionSwitch } from "@/components/action/ActionSwitch";
import { useFetchIssueSyncs, useSetIssueSync } from "@/hooks/IssueSyncHooks";
import { syncOf } from "@/models/IssueSync";
import type { RepoRef } from "@/models/Project";

interface RepoIssuesSwitchProps {
  projectId: string;
  repo: RepoRef;
}

// Links the repository's labelled GitHub issues to tickets; off, nothing on GitHub changes a ticket.
export const RepoIssuesSwitch = ({ projectId, repo }: RepoIssuesSwitchProps) => {
  const { data: syncs } = useFetchIssueSyncs(projectId);
  const set = useSetIssueSync(projectId);
  const sync = syncOf(syncs, repo.owner, repo.name);
  const label = `Issues of ${repo.full_name}`;
  return (
    <span className="flex items-center gap-2 font-sans">
      <span aria-hidden className="text-xs text-muted-foreground">
        Issues
      </span>
      <ActionSwitch
        permission="projects:write"
        projectId={projectId}
        verb="change repositories"
        aria-label={label}
        size="sm"
        checked={set.isPending ? set.variables.enabled : (sync?.enabled ?? false)}
        disabled={!syncs || set.isPending}
        onCheckedChange={(enabled) => set.mutate({ owner: repo.owner, repo: repo.name, enabled })}
      />
    </span>
  );
};
