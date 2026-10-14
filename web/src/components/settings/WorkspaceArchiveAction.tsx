import { DangerAction, DangerButton } from "@/components/settings/DangerZone";
import { WorkspaceArchiveDetails } from "@/components/settings/WorkspaceArchiveDetails";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useArchiveWorkspace } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";

interface WorkspaceArchiveActionProps {
  workspace: Workspace;
}

export const WorkspaceArchiveAction = ({ workspace }: WorkspaceArchiveActionProps) => {
  const archive = useArchiveWorkspace();
  const { data: people } = useFetchWorkspacePeople(workspace.id);
  const { data: projects } = useFetchWorkspaceProjects(workspace.id);
  const { open: confirmArchive } = useConfirmationDialog();

  const onArchive = async () => {
    const ok = await confirmArchive({
      title: `Archive ${workspace.name}?`,
      message: "It leaves every list, search and live update until the instance Owner restores it. Nothing is deleted.",
      confirmLabel: "Archive workspace",
      details: <WorkspaceArchiveDetails people={people?.length} projects={projects?.length} />,
    });
    if (ok) archive.mutate(workspace.id);
  };

  return (
    <DangerAction
      title="Archive workspace"
      consequence="Hides this workspace from everyone in it. The instance Owner restores it from Settings → Instance."
      action={
        <DangerButton loading={archive.isPending} onClick={() => void onArchive()}>
          Archive workspace
        </DangerButton>
      }
    />
  );
};
