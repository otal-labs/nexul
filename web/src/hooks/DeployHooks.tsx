import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { appendLogLines, lastLogSeq } from "@nexul/client-core/deployLog";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { Deploy, DeployLogLine, DeployStatus } from "@/models/Stack";
import type { LiveFollower } from "@/lib/live";

export const getDeployKey = "getDeploy";
export const getDeployLogKey = "getDeployLog";

export const useFetchDeploy = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployKey, id],
    queryFn: async () => (await api.get<Deploy>(`/api/deploys/${id}`)).data,
    enabled: !!id,
  });

export const useFetchDeployLog = (id: string | undefined) =>
  useQuery({
    queryKey: [getDeployLogKey, id],
    queryFn: async () => (await api.get<DeployLogLine[]>(`/api/deploys/${id}/log`)).data,
    enabled: !!id,
  });

export const useCancelDeploy = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("deploys:write", async (id: string) => api.post(`/api/deploys/${id}/cancel`)),
    onSuccess: async (_data, id) => {
      await client.invalidateQueries({ queryKey: [getDeployKey, id] });
      toast.success("Cancel requested");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A deploy.updated frame: the record's status, or its log, changed (internal/deploy/events.go).
export interface DeployFrame {
  id: string;
  status: DeployStatus;
  stack_id: string;
}

// A log batch leaves the history row as it was, so the stack's history refetches only for a new deploy or a new status.
export const refetchDeployHistory = (client: QueryClient, key: string, { id, status, stack_id }: DeployFrame) => {
  const queryKey = [key, stack_id];
  if (client.getQueryData<Deploy[]>(queryKey)?.find((d) => d.id === id)?.status === status) return;
  return client.invalidateQueries({ queryKey, exact: true });
};

// An open log fetches only the lines after its newest one, up to four batches a second; a closed one loads whole on open.
const appendDeployLog = async (client: QueryClient, id: string) => {
  const queryKey = [getDeployLogKey, id];
  const cached = client.getQueryData<DeployLogLine[]>(queryKey);
  if (!cached || !client.getQueryCache().find({ queryKey, exact: true })?.isActive()) return client.invalidateQueries({ queryKey, exact: true });
  try {
    const { data } = await api.get<DeployLogLine[]>(`/api/deploys/${id}/log?after=${lastLogSeq(cached)}`);
    client.setQueryData<DeployLogLine[]>(queryKey, (lines) => appendLogLines(lines ?? [], data));
  } catch {
    await client.invalidateQueries({ queryKey, exact: true });
  }
};

export const deployFollower: LiveFollower = {
  // Only deploy.updated refetches the record; the runner's topics fire before the deploy domain commits the change.
  "deploy.updated": ({ id }: DeployFrame, { client }) =>
    Promise.all([client.invalidateQueries({ queryKey: [getDeployKey, id], exact: true }), appendDeployLog(client, id)]),
};
