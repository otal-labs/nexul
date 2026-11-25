import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { appendLogLines, lastLogSeq } from "@nexul/client-core/deployLog";
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

// An open log fetches only the lines after its newest one, up to four batches a second; a closed one loads whole on open.
const appendDeployLog = async (client: QueryClient, id: string | undefined) => {
  if (!id) return client.invalidateQueries({ queryKey: [getDeployLogKey] });
  const queryKey = [getDeployLogKey, id];
  const cached = client.getQueryData<DeployLogLine[]>(queryKey);
  if (!cached || !client.getQueryCache().find({ queryKey, exact: true })?.isActive()) return client.invalidateQueries({ queryKey, exact: true });
  try {
    const added = await api.get<DeployLogLine[]>(`/api/deploys/${id}/log?after=${lastLogSeq(cached)}`);
    client.setQueryData<DeployLogLine[]>(queryKey, (lines) => appendLogLines(lines ?? [], added));
  } catch {
    await client.invalidateQueries({ queryKey, exact: true });
  }
};

// The deploy screen's log follows the tail off this alone: each batch the frames name is appended to the cached lines.
const deployLogQuery = defineQuery({
  key: getDeployLogKey,
  fetch: (id: string | undefined) => api.get<DeployLogLine[]>(`/api/deploys/${id}/log`),
  refreshes: { "deploy.updated": { patch: (client, p) => void appendDeployLog(client, p.id) } },
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
