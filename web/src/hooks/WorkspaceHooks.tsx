import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { hasPermission } from "@/models/Permission";
import type { Workspace } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const getWorkspacesKey = "getWorkspaces";

export const useFetchWorkspaces = (enabled = true) =>
  useQuery({
    queryKey: [getWorkspacesKey],
    queryFn: async () => (await api.get<Workspace[]>("/api/workspaces")).data,
    enabled,
  });

// F5 exception: repairs an empty or stale selection so every workspace-scoped query works even where no switcher renders (onboarding). getState() avoids clobbering a just-created id before refetch.
export const useEnsureWorkspaceSelected = (enabled: boolean) => {
  const { data: workspaces } = useFetchWorkspaces(enabled);
  useEffect(() => {
    if (!workspaces) return;
    const { selectedWorkspaceId, selectWorkspace } = useWorkspaceStore.getState();
    const stillMember = workspaces.some((w) => w.id === selectedWorkspaceId);
    if (!stillMember) selectWorkspace(workspaces[0]?.id ?? "");
  }, [workspaces]);
};

const getMyRoleKey = "getMyRole";

// Mirrors internal/tenancy/handler.go's meResponse; keep the two in sync.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
}

// Keyed by workspaceId so switching workspaces refetches; multiple call sites share the cache via dedupe.
export const useFetchMyRole = (workspaceId: string) =>
  useQuery({
    queryKey: [getMyRoleKey, workspaceId],
    queryFn: async () => (await api.get<MyWorkspaceInfo>(`/api/workspaces/${workspaceId}/me`)).data,
    enabled: workspaceId !== "",
  });

// The frontend's single hasPermission(value) helper; no call site should compute permissions itself.
export const useHasPermission = (value: string): boolean => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data } = useFetchMyRole(selectedWorkspaceId);
  return hasPermission(data?.permissions, value);
};

// No success toast: the owner wizard batches this with other finish-step mutations and shows its own confirmation.
export const useRenameWorkspace = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { id: string; name: string }) =>
      (await api.patch<Workspace>(`/api/workspaces/${payload.id}`, { name: payload.name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacesKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCreateWorkspace = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => (await api.post<Workspace>("/api/workspaces", { name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacesKey] });
      toast.success("Workspace created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
