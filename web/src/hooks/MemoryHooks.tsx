import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { CreateMemoryFormData, Memory } from "@/models/Memory";
import type { MemoryVersion } from "@/models/MemoryVersion";
import type { Project } from "@/models/Project";
import type { Workspace } from "@/models/Workspace";

export const getMemoriesKey = "getMemories";
export const getMemoryKey = "getMemory";
export const getMemoryVersionsKey = "getMemoryVersions";
const getProjectsKeyForClone = "getProjectsForClone";

// Workspace-wide list for the Memories page; the page groups these by project client-side.
export const useFetchMemories = (workspaceId: string) =>
  useQuery({
    queryKey: [getMemoriesKey, "byWorkspace", workspaceId],
    queryFn: async () =>
      (await api.get<Memory[]>("/api/memories", { params: { workspace_id: workspaceId } })).data,
    enabled: workspaceId !== "",
  });

export const useFetchMemoriesByProject = (projectId: string) =>
  useQuery({
    queryKey: [getMemoriesKey, "byProject", projectId],
    queryFn: async () => (await api.get<Memory[]>("/api/memories", { params: { project_id: projectId } })).data,
    enabled: projectId !== "",
  });

export const useFetchMemory = (id: string | undefined) =>
  useQuery({
    queryKey: [getMemoryKey, id],
    queryFn: async () => (await api.get<Memory>(`/api/memories/${id}`)).data,
    enabled: !!id,
  });

export const useCreateMemory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateMemoryFormData) => (await api.post<Memory>("/api/memories", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMemoriesKey] });
      toast.success("Memory created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface UpdateMemoryInput {
  id: string;
  title: string;
  when_to_use: string;
  body: string;
  always_included: boolean;
}

export const useUpdateMemory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...input }: UpdateMemoryInput) =>
      (await api.put<Memory>(`/api/memories/${id}`, input)).data,
    onSuccess: async (_, { id }) => {
      await client.invalidateQueries({ queryKey: [getMemoriesKey] });
      await client.invalidateQueries({ queryKey: [getMemoryKey, id] });
      toast.success("Memory saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteMemory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/memories/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMemoriesKey] });
      toast.success("Memory deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchMemoryVersions = (id: string | undefined) =>
  useQuery({
    queryKey: [getMemoryVersionsKey, id],
    queryFn: async () => (await api.get<MemoryVersion[]>(`/api/memories/${id}/versions`)).data,
    enabled: !!id,
  });

export const useRevertMemory = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, version }: { id: string; version: number }) =>
      (await api.post<Memory>(`/api/memories/${id}/revert`, { version })).data,
    onSuccess: async (_, { id }) => {
      await client.invalidateQueries({ queryKey: [getMemoryKey, id] });
      await client.invalidateQueries({ queryKey: [getMemoryVersionsKey, id] });
      toast.success("Memory reverted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCloneMemory = () => {
  const client = useQueryClient();
  return useMutation({
    // projectId "" clones to workspace scope; workspaceId always names the destination workspace.
    mutationFn: async ({ id, projectId, workspaceId }: { id: string; projectId: string; workspaceId: string }) =>
      (await api.post<Memory>(`/api/memories/${id}/clone`, { project_id: projectId, workspace_id: workspaceId }))
        .data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMemoriesKey] });
      toast.success("Memory cloned");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export interface CloneDestination {
  workspace: Workspace;
  projects: Project[];
}

// Every workspace the user belongs to, each with its own projects, for the "Clone to…" picker — grouped
// client-side since /api/projects only ever answers for one workspace at a time.
export const useFetchCloneDestinations = () => {
  const workspaces = useFetchWorkspaces();
  const workspaceList = workspaces.data ?? [];
  const results = useQueries({
    queries: workspaceList.map((w) => ({
      queryKey: [getProjectsKeyForClone, w.id],
      queryFn: async () => (await api.get<Project[]>("/api/projects", { params: { workspace_id: w.id } })).data,
      enabled: !!workspaces.data,
    })),
  });

  const isPending = workspaces.isPending || results.some((r) => r.isPending);
  const error = workspaces.error ?? results.find((r) => r.error)?.error ?? null;
  const data: CloneDestination[] | undefined =
    workspaces.data && results.every((r) => r.data)
      ? workspaceList.map((workspace, i) => ({ workspace, projects: results[i]?.data ?? [] }))
      : undefined;

  return { data, isPending, error };
};
