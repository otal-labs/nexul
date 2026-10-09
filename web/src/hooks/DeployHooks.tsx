import { useMutation, useQuery, useQueryClient, type QueryClient, type QueryKey } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Deploy, DeployLogLine } from "@/models/Stack";
import { refetchHolding, type LiveFollower } from "@/lib/live";

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
    mutationFn: async (id: string) => api.post(`/api/deploys/${id}/cancel`),
    onSuccess: async (_data, id) => {
      await client.invalidateQueries({ queryKey: [getDeployKey, id] });
      toast.success("Cancel requested");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The frame names no stack: the histories holding the deploy refetch, or all of them for a deploy none holds yet.
export const refetchDeployHistories = (client: QueryClient, queryKey: QueryKey, id: string) => {
  const holds = (deploys: Deploy[]) => deploys.some((d) => d.id === id);
  const held = client.getQueriesData<Deploy[]>({ queryKey }).some(([, deploys]) => deploys && holds(deploys));
  return held ? refetchHolding(client, queryKey, holds) : client.invalidateQueries({ queryKey });
};

export const deployFollower: LiveFollower = {
  // Only deploy.updated refetches the record; the runner's topics fire before the deploy domain commits the change.
  // ponytail: whole-log refetch per batch (≤ 4/s); append lines into the cache if logs get large.
  "deploy.updated": ({ id }: { id: string }, { client }) =>
    Promise.all([
      client.invalidateQueries({ queryKey: [getDeployKey, id], exact: true }),
      client.invalidateQueries({ queryKey: [getDeployLogKey, id], exact: true }),
    ]),
};
