import { useEffect } from "react";
import { Outlet, useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ErrorScreen } from "@/components/ErrorScreen";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The URL names the workspace and the store follows it, so everything reading the selection sees the one in view;
// a slug the viewer isn't a member of is not found.
export const WorkspaceScope = () => {
  const { workspace: slug } = useParams();
  const { data: workspaces, error, isPending } = useFetchWorkspaces();
  const workspace = workspaces?.find((w) => w.slug === slug);
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedWorkspaceSlug = useWorkspaceStore((s) => s.selectedWorkspaceSlug);
  const selectWorkspace = useWorkspaceStore((s) => s.selectWorkspace);
  const inSync = workspace !== undefined && workspace.id === selectedWorkspaceId && workspace.slug === selectedWorkspaceSlug;

  // F5 exception: one-way sync from the URL into the store; nothing writes the URL from the store.
  useEffect(() => {
    if (workspace && !inSync) selectWorkspace(workspace.id, workspace.slug);
  }, [workspace, inSync, selectWorkspace]);

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {workspaces && !workspace && <ErrorScreen />}
      {inSync && <Outlet />}
    </>
  );
};
