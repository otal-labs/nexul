import { useQuery } from "@tanstack/react-query";

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
