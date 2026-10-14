import { DangerZone } from "@/components/settings/DangerZone";
import { WorkspaceArchiveAction } from "@/components/settings/WorkspaceArchiveAction";
import { EmptyRow } from "@/components/EmptyRow";
import { useSelectedWorkspace, useHasPermission } from "@/hooks/WorkspaceHooks";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

// A project is removed from its own settings; the one workspace-wide action is archiving, which the instance's own workspace can't take.
export const DangerZoneSection = () => {
  const workspace = useSelectedWorkspace();
  const canArchive = useHasPermission("workspaces:delete");
  const archivable = workspace !== undefined && canArchive && workspace.id !== INSTANCE_WORKSPACE_ID;
  return (
    <DangerZone>
      {archivable && <WorkspaceArchiveAction workspace={workspace} />}
      {!archivable && (
        <li className="py-3">
          <EmptyRow flush>Nothing here can't be undone yet. To remove a project, open its settings.</EmptyRow>
        </li>
      )}
    </DangerZone>
  );
};
