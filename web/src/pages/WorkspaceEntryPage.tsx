import { Navigate } from "react-router";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchWorkspaces, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { HomePage } from "@/pages/HomePage";
import { workspacePath } from "@/models/Workspace";

// "/" signed in: the last workspace in view, or the first one, is where the app opens.
export const WorkspaceEntryPage = () => {
  const { data: workspaces, isPending } = useFetchWorkspaces();
  const target = useSelectedWorkspace() ?? workspaces?.[0];

  return (
    <>
      {isPending && <LoadingDisplay />}
      {target && <Navigate to={workspacePath(target.slug, "/")} replace />}
      {workspaces && workspaces.length === 0 && <HomePage />}
    </>
  );
};
