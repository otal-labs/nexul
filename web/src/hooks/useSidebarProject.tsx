import { useEffect } from "react";
import { useLocation } from "react-router";

import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { projectTokenFromPath, resolveProject } from "@/models/Project";

// The project the sidebar shows: the one the URL is on, else the last one visited, else the first.
export const useSidebarProject = () => {
  const { data: projects } = useFetchProjects();
  const { pathname } = useLocation();
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const selectProject = useWorkspaceStore((s) => s.selectProject);

  const token = projectTokenFromPath(pathname);
  const routed = token && projects ? resolveProject(projects, token) : undefined;
  const current = routed ?? projects?.find((p) => p.id === selectedProjectId) ?? projects?.[0];
  const routedId = routed?.id;

  // Remembered so leaving for Runners or Chat keeps the sidebar on the project just visited.
  useEffect(() => {
    if (routedId) selectProject(routedId);
  }, [routedId, selectProject]);

  return { projects, current };
};
