import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Project } from "@/models/Project";

const getWorkspacesKey = "getWorkspaces";
export const getProjectsKey = "getProjects";
export const getProjectKey = "getProject";

// There is no workspace switcher on the phone yet (ticket 11: that lives in Your settings), so the board
// works against the first workspace the signed-in member belongs to.
export const useCurrentWorkspaceId = (): string | undefined => {
  const { data } = useQuery({
    queryKey: [getWorkspacesKey],
    queryFn: () => api.get<{ id: string }[]>("/api/workspaces"),
  });
  return data?.[0]?.id;
};

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
