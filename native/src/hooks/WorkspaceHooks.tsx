import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Workspace } from "@/models/Workspace";

export const getWorkspacesKey = "getWorkspaces";

export const useFetchWorkspaces = () =>
  useQuery({
    queryKey: [getWorkspacesKey],
    queryFn: () => api.get<Workspace[]>("/api/workspaces"),
  });
