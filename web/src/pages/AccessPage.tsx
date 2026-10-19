import { useEffect } from "react";

import { AccessRolesView } from "@/components/accessgrid/AccessRolesView";
import { AccessSetupView } from "@/components/accessgrid/AccessSetupView";
import { AccessTabsView } from "@/components/accessgrid/AccessTabsView";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useManagesAccess } from "@/hooks/AccessHooks";
import { useAccessSetupState, useFetchAccessGrid, useIsWorkspaceOwner } from "@/hooks/AccessGridHooks";
import { useAccessDraftStore } from "@/stores/accessDraftStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Who's in each project, who sees each public channel and what each role can do: the Owner's one-time setup, then the workspace's Access page.
export const AccessPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const managesAccess = useManagesAccess();
  const { data: grid, error, isPending } = useFetchAccessGrid(workspaceId, managesAccess === true);
  const settingUp = useAccessSetupState() === "owner";
  const owner = useIsWorkspaceOwner();
  // Ticks held on one workspace's page never carry to another, or to the next visit.
  useEffect(() => useAccessDraftStore.getState().clear, [workspaceId]);

  return (
    <Container className="py-8">
      {managesAccess && isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {grid && settingUp && <AccessSetupView grid={grid} />}
      {grid && !settingUp && <AccessTabsView grid={grid} owner={owner} />}
      {managesAccess === false && <AccessRolesView />}
    </Container>
  );
};
