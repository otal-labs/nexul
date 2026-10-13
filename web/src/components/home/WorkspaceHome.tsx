import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoRolesState } from "@/components/home/NoRolesState";
import { WorkspaceWelcome } from "@/components/home/WorkspaceWelcome";
import { useAccessSetupRedirect } from "@/hooks/useAccessSetupRedirect";
import { useFetchMyRole, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// A member with no role meets why the workspace is empty and who can change that, never a page of missing links.
export const WorkspaceHome = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const workspace = useSelectedWorkspace();
  const { data: me, isPending } = useFetchMyRole(workspaceId);
  const noRoles = me?.roles?.length === 0;
  useAccessSetupRedirect();

  return (
    <>
      {workspaceId && isPending && <LoadingDisplay />}
      {workspace && noRoles && <NoRolesState workspace={workspace} />}
      {(!workspaceId || (me && !noRoles)) && <WorkspaceWelcome />}
    </>
  );
};
