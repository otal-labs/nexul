import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import { latestDeploy, type Container, type Deploy, type Stack } from "@/models/Stack";

export const getStacksKey = "getStacks";
export const getStackKey = "getStack";
export const getStackServicesKey = "getStackServices";
export const getStackDeploysKey = "getStackDeploys";

export const useFetchStacks = () =>
  useQuery({
    queryKey: [getStacksKey],
    queryFn: () => api.get<Stack[]>("/api/stacks"),
  });

export const useFetchStack = (id: string | undefined) =>
  useQuery({
    queryKey: [getStackKey, id],
    queryFn: () => api.get<Stack>(`/api/stacks/${id}`),
    enabled: !!id,
  });

export const useFetchStackServices = (stackId: string | undefined) =>
  useQuery({
    queryKey: [getStackServicesKey, stackId],
    queryFn: () => api.get<Container[]>(`/api/stacks/${stackId}/services`),
    enabled: !!stackId,
  });

export const useFetchStackDeploys = (stackId: string | undefined) =>
  useQuery({
    queryKey: [getStackDeploysKey, stackId],
    queryFn: () => api.get<Deploy[]>(`/api/stacks/${stackId}/deploys`),
    enabled: !!stackId,
  });

export interface StackWithLatestDeploy {
  stack: Stack;
  latest: Deploy | undefined;
}

// The stack list's status dot and last-deploy time need each stack's latest deploy, which /api/stacks doesn't
// carry; this fans out one deploy-history query per stack, the same shape as the web topology canvas's
// per-stack container fetch (useFetchAllContainers).
export const useFetchStacksWithLatestDeploy = () => {
  const stacks = useFetchStacks();
  const stackIds = stacks.data?.map((s) => s.id) ?? [];
  const results = useQueries({
    queries: stackIds.map((id) => ({
      queryKey: [getStackDeploysKey, id],
      queryFn: () => api.get<Deploy[]>(`/api/stacks/${id}/deploys`),
      enabled: !!stacks.data,
    })),
  });

  const isPending = stacks.isPending || results.some((r) => r.isPending);
  const error = stacks.error ?? results.find((r) => r.error)?.error;
  const data: StackWithLatestDeploy[] | undefined =
    stacks.data && results.every((r) => r.data)
      ? stacks.data.map((stack, i) => ({ stack, latest: latestDeploy(results[i]?.data) }))
      : undefined;

  return { data, isPending, error };
};

// The runner's pre-built-image path only knows the run strategy: it removes the container and runs the image
// again. Build-from-ref stays web-only for now (ticket 11: "Redeploy behind a confirm", not a build form).
export const useDeployStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ stackId, image }: { stackId: string; image: string }) =>
      api.post<Deploy>("/api/deploys", { stack_id: stackId, image }),
    onSuccess: async (_data, vars) => {
      await client.invalidateQueries({ queryKey: [getStackDeploysKey, vars.stackId] });
    },
  });
};
