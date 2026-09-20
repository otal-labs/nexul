import { useEffect } from "react";
import { useNavigate } from "react-router";

import { boardPath, resolveProject } from "@/models/Project";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface UseBoardProjectScopeArgs {
  routeParam: string | undefined;
  projects: Project[];
  isLoading: boolean;
  error: unknown;
}

// Resolves the board's scoped project from the URL and redirects the unscoped route to the last-viewed one.
export const useBoardProjectScope = ({ routeParam, projects, isLoading, error }: UseBoardProjectScopeArgs) => {
  const navigate = useNavigate();
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const selectProject = useWorkspaceStore((s) => s.selectProject);

  // The URL token is the project's prefix or (old links) its id; unresolvable gets a not-found state.
  const scopedProject = routeParam ? resolveProject(projects, routeParam) : undefined;
  const scopedProjectId = scopedProject?.id;
  const scopedNotFound = Boolean(routeParam) && !isLoading && !error && !scopedProject;
  const redirectProject = routeParam
    ? undefined
    : (projects.find((p) => p.id === selectedProjectId) ?? projects[0]);

  useEffect(() => {
    if (routeParam || isLoading || !redirectProject) return;
    navigate(boardPath(redirectProject), { replace: true });
  }, [routeParam, isLoading, redirectProject, navigate]);

  // Tracks the last-viewed project so the next unscoped /board visit redirects back here.
  useEffect(() => {
    if (scopedProjectId) selectProject(scopedProjectId);
  }, [scopedProjectId, selectProject]);

  return { scopedProject, scopedProjectId, scopedNotFound };
};
