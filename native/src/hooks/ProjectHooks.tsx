import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { api } from "@/api/client";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";

export const getProjectsKey = "getProjects";
export const getProjectKey = "getProject";

export const useFetchProjects = () => {
  const workspaceId = useCurrentWorkspaceId();
  return useQuery({
    queryKey: [getProjectsKey, workspaceId],
    queryFn: () => api.get<Project[]>(`/api/projects?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
    enabled: !!workspaceId,
  });
};

export const useFetchProject = (id: string | undefined) =>
  useQuery({
    queryKey: [getProjectKey, id],
    queryFn: () => api.get<Project>(`/api/projects/${id}`),
    enabled: !!id,
  });

// True once a project this screen showed leaves the viewer's list; one it never showed stays the screen's own not found.
export const useRevokedProject = (projectId: string | undefined): boolean => {
  const { data: projects } = useFetchProjects();
  const [opened, setOpened] = useState<string>();
  const found = !!projectId && !!projects?.some((project) => project.id === projectId);
  if (found && opened !== projectId) setOpened(projectId);
  return !!projectId && !!projects && !found && opened === projectId;
};
