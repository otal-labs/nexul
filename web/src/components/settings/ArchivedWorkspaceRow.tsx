import { Button } from "@/components/ui/button";
import { useRestoreWorkspace } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface ArchivedWorkspaceRowProps {
  workspace: Workspace;
}

export const ArchivedWorkspaceRow = ({ workspace }: ArchivedWorkspaceRowProps) => {
  const restore = useRestoreWorkspace();
  return (
    <li className="flex items-center gap-3 px-3 py-2.5">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{workspace.name}</p>
        <p className="truncate text-xs text-muted-foreground">
          <span className="font-mono">/{workspace.slug}</span>
          {workspace.archived_at && <> · archived {formatRelativeTime(workspace.archived_at)}</>}
        </p>
      </div>
      <Button variant="outline" size="sm" loading={restore.isPending} onClick={() => restore.mutate(workspace.id)}>
        Restore
      </Button>
    </li>
  );
};
