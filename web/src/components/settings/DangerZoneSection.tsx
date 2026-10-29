import { DangerZone } from "@/components/settings/DangerZone";
import { WorkspaceArchiveAction } from "@/components/settings/WorkspaceArchiveAction";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

// Archiving is the one workspace-wide action, shown off for the instance's own workspace; a role that hides it has no Danger zone.
export const DangerZoneSection = () => {
  const workspace = useSelectedWorkspace();
  const own = workspace?.id === INSTANCE_WORKSPACE_ID;
  return (
    <DangerZone title="This workspace">
      {workspace && <WorkspaceArchiveAction workspace={workspace} blocked={own ? "The instance's own workspace can't be archived." : undefined} />}
    </DangerZone>
  );
};
