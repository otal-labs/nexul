import { useQuery } from "@tanstack/react-query";

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
