import { useState } from "react";

import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePathname } from "@/hooks/useWorkspacePath";
import { openProjectToken, resolveProject } from "@/models/Project";

// True once the project a page shows drops out of the viewer's project list while the page is open. A link to a
// project they never saw is plain not found, so only a token that resolved on this page counts.
export const useRevokedProject = (): boolean => {
  const token = openProjectToken(useWorkspacePathname())?.toLowerCase();
  const { data: projects } = useFetchProjects();
  const [opened, setOpened] = useState<string>();
  const found = !!token && !!projects && !!resolveProject(projects, token);
  if (found && opened !== token) setOpened(token);
  return !!token && !!projects && !found && opened === token;
};
