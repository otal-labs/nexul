import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { latestDeploy, type Container, type CreateStackInput, type CreateStackResponse, type Deploy, type Stack, type StackWithBranches } from "@/models/Stack";

export const getStacksKey = "getStacks";
export const getStackKey = "getStack";
export const getStackServicesKey = "getStackServices";
export const getStackDeploysKey = "getStackDeploys";

// No projectId lists every project's stacks, which the workspace-wide topology canvas wires from.
export const useFetchStacks = (projectId?: string) =>
  useQuery({
    queryKey: [getStacksKey, projectId ?? "all"],
    queryFn: async () =>
      (await api.get<Stack[]>("/api/stacks", { params: projectId ? { project_id: projectId } : {} })).data,
  });

export const useFetchStack = (id: string | undefined) =>
  useQuery({
    queryKey: [getStackKey, id],
    queryFn: async () => (await api.get<StackWithBranches>(`/api/stacks/${id}`)).data,
    enabled: !!id,
  });

export const useFetchStackServices = (stackId: string | undefined) =>
  useQuery({
    queryKey: [getStackServicesKey, stackId],
    queryFn: async () => (await api.get<Container[]>(`/api/stacks/${stackId}/services`)).data,
    enabled: !!stackId,
  });

// pollMs keeps refetching while a deploy is pending/running, for watchers that cannot rely on catching the live event.
export const useFetchStackDeploys = (stackId: string | undefined, pollMs?: number) =>
  useQuery({
    queryKey: [getStackDeploysKey, stackId],
    queryFn: async () => (await api.get<Deploy[]>(`/api/stacks/${stackId}/deploys`)).data,
    enabled: !!stackId,
    refetchInterval: (query) => {
      if (!pollMs) return false;
      const latest = latestDeploy(query.state.data);
      return latest?.status === "healthy" || latest?.status === "failed" ? false : pollMs;
    },
  });

// Every field on Stack is required by the backend's full-replace PATCH (spec: no partial merge), so callers pass
// the whole record with their change applied, not a partial payload.
export const useUpdateStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (stack: Stack) => (await api.patch<Stack>(`/api/stacks/${stack.id}`, stack)).data,
    onSuccess: async (_data, stack) => {
      await client.invalidateQueries({ queryKey: [getStackKey, stack.id] });
      await client.invalidateQueries({ queryKey: [getStacksKey] });
      toast.success("Stack updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/stacks/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStacksKey] });
      toast.success("Stack removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeployStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ stackId, image, ref }: { stackId: string; image?: string; ref?: string }) =>
      (await api.post<Deploy>("/api/deploys", { stack_id: stackId, image, ref })).data,
    onSuccess: async (_data, vars) => {
      await client.invalidateQueries({ queryKey: [getStackDeploysKey, vars.stackId] });
      toast.success("Deploy triggered");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRollbackStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (stackId: string) => (await api.post<Deploy>(`/api/stacks/${stackId}/rollback`)).data,
    onSuccess: async (_data, stackId) => {
      await client.invalidateQueries({ queryKey: [getStackDeploysKey, stackId] });
      toast.success("Rollback triggered");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Workspace-wide containers for the topology canvas: no single "all containers" endpoint exists (containers are
// listed per stack), so this fans out one request per stack once the stack list is known.
export const useFetchAllContainers = () => {
  const stacks = useFetchStacks();
  const stackIds = stacks.data?.map((s) => s.id) ?? [];
  const results = useQueries({
    queries: stackIds.map((id) => ({
      queryKey: [getStackServicesKey, id],
      queryFn: async () => (await api.get<Container[]>(`/api/stacks/${id}/services`)).data,
      enabled: !!stacks.data,
    })),
  });

  const isPending = stacks.isPending || results.some((r) => r.isPending);
  const error = stacks.error ?? results.find((r) => r.error)?.error ?? null;
  const data = results.every((r) => r.data) ? results.flatMap((r) => r.data ?? []) : undefined;

  return { data, isPending, error };
};

export const useUpdateStackEnv = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (stack: Stack) => (await api.patch<Stack>(`/api/stacks/${stack.id}`, stack)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getStackKey] });
      toast.success("Environment saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// deploy: true (plus declared/link_repository) makes this a one-call "create & deploy"; the 201 body carries the
// stack plus a `deploy` ref when that flag was set.

export const useCreateStack = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateStackInput) => (await api.post<CreateStackResponse>("/api/stacks", input)).data,
    onSuccess: async () => {
      // /api/services stays the wire alias for one release (internal/deploy/handler.go); the old list key
      // still backs ProjectServices until that page moves to stacks.
      await client.invalidateQueries({ queryKey: [getStacksKey] });
      toast.success("Stack created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
