import { useEffect } from "react";

import { AccessSetupView } from "@/components/accessgrid/AccessSetupView";
import { AccessTabsView } from "@/components/accessgrid/AccessTabsView";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useAccessSetupState, useFetchAccessGrid, useIsWorkspaceOwner } from "@/hooks/AccessGridHooks";
import { useAccessDraftStore } from "@/stores/accessDraftStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Who's in each project and who sees each public channel: the Owner's one-time setup, then the workspace's Access page.
export const AccessPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: grid, error, isPending } = useFetchAccessGrid(workspaceId);
  const settingUp = useAccessSetupState() === "owner";
  const owner = useIsWorkspaceOwner();
  // Ticks held on one workspace's page never carry to another, or to the next visit.
  useEffect(() => useAccessDraftStore.getState().clear, [workspaceId]);

  return (
    <Container className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {grid && settingUp && <AccessSetupView grid={grid} />}
      {grid && !settingUp && <AccessTabsView grid={grid} owner={owner} />}
    </Container>
  );
};
