import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { defineQuery } from "@/lib/liveQuery";
import { latestDeploy, type Container, type Deploy, type Stack } from "@/models/Stack";

export const getStacksKey = "getStacks";
export const getStackKey = "getStack";
export const getStackServicesKey = "getStackServices";
export const getStackDeploysKey = "getStackDeploys";

const stacksQuery = defineQuery({
  key: getStacksKey,
  fetch: (workspaceId: string | undefined) =>
    api.get<Stack[]>(`/api/stacks?workspace_id=${encodeURIComponent(workspaceId ?? "")}`),
  refreshes: {},
});

export const useFetchStacks = () => {
  const workspaceId = useCurrentWorkspaceId();
  return useQuery({ ...stacksQuery.options(workspaceId), enabled: !!workspaceId });
};

const stackQuery = defineQuery({
  key: getStackKey,
  fetch: (id: string | undefined) => api.get<Stack>(`/api/stacks/${id}`),
  refreshes: {},
});

export const useFetchStack = (id: string | undefined) => useQuery({ ...stackQuery.options(id), enabled: !!id });

const stackServicesQuery = defineQuery({
  key: getStackServicesKey,
  fetch: (stackId: string | undefined) => api.get<Container[]>(`/api/stacks/${stackId}/services`),
  refreshes: {},
});

export const useFetchStackServices = (stackId: string | undefined) =>
  useQuery({ ...stackServicesQuery.options(stackId), enabled: !!stackId });

// A deploy frame names the deploy, not its stack, so every stack's history refetches.
const stackDeploysQuery = defineQuery({
  key: getStackDeploysKey,
  fetch: (stackId: string | undefined) => api.get<Deploy[]>(`/api/stacks/${stackId}/deploys`),
  refreshes: { "deploy.updated": "all" },
});

export const useFetchStackDeploys = (stackId: string | undefined) =>
  useQuery({ ...stackDeploysQuery.options(stackId), enabled: !!stackId });

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
    queries: stackIds.map((id) => ({ ...stackDeploysQuery.options(id), enabled: !!stacks.data })),
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
    mutationFn: async ({ stackId, image }: { stackId: string; image: string }) => {
      try {
        return await api.post<Deploy>("/api/deploys", { stack_id: stackId, image });
      } catch (error) {
        if (error instanceof ApiError && error.status === 409) {
          throw new Error("A deploy is already running on this stack. Redeploy once it finishes.");
        }
        throw error;
      }
    },
    onSuccess: async (_data, vars) => {
      await client.invalidateQueries({ queryKey: [getStackDeploysKey, vars.stackId] });
    },
  });
};
