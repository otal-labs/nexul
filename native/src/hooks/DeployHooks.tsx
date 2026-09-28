import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Deploy, DeployLogLine } from "@/models/Stack";

export const getDeployKey = "getDeploy";
export const getDeployLogKey = "getDeployLog";

export const useFetchDeploy = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployKey, id],
    queryFn: () => api.get<Deploy>(`/api/deploys/${id}`),
    enabled: !!id,
  });

export const useFetchDeployLog = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployLogKey, id],
    queryFn: () => api.get<DeployLogLine[]>(`/api/deploys/${id}/log`),
    enabled: !!id,
  });
