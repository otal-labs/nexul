import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Deploy, DeployLogLine } from "@/models/Stack";

export const getDeployKey = "getDeploy";
export const getDeployLogKey = "getDeployLog";

const deployQuery = defineQuery({
  key: getDeployKey,
  fetch: (id: string | undefined) => api.get<Deploy>(`/api/deploys/${id}`),
  refreshes: { "deploy.updated": { record: (p) => p.id } },
});

export const useFetchDeploy = (id: string | undefined) => useQuery({ ...deployQuery.options(id), enabled: !!id });

// The deploy screen's log follows the tail off this alone: a refetch here is what turns into new log lines.
const deployLogQuery = defineQuery({
  key: getDeployLogKey,
  fetch: (id: string | undefined) => api.get<DeployLogLine[]>(`/api/deploys/${id}/log`),
  refreshes: { "deploy.updated": { record: (p) => p.id } },
});

export const useFetchDeployLog = (id: string | undefined) => useQuery({ ...deployLogQuery.options(id), enabled: !!id });

// The runner stops the deploy and the live deploy.updated frame carries the new status; this only asks.
export const useCancelDeploy = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("deploys:write", (id: string) => api.post<null>(`/api/deploys/${id}/cancel`)),
    onSuccess: (_result, id) => client.invalidateQueries({ queryKey: [getDeployKey, id] }),
  });
};
