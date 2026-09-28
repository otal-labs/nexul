import { Navigate } from "react-router";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { NEW_PROJECT_PATH } from "@/models/Project";

// Someone who can create projects, landing in a workspace with none, starts in the project wizard; everyone
// else sees the empty states, which point there too.
export const FirstProjectRedirect = () => {
  const inWorkspace = useWorkspaceStore((s) => s.selectedWorkspaceId) !== "";
  const { data: me } = useFetchMe();
  const { data: projects } = useFetchProjects(inWorkspace);
  const canCreate = me?.user?.can_create_workspace ?? false;

  return canCreate && projects && projects.length === 0 && <Navigate to={NEW_PROJECT_PATH} replace />;
};
