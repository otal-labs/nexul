import { useEffect } from "react";
import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { hasPermission } from "@/models/Permission";
import type { Workspace } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getWorkspacesKey = "getWorkspaces";

export const useFetchWorkspaces = (enabled = true) =>
  useQuery({
    queryKey: [getWorkspacesKey],
    queryFn: async () => (await api.get<Workspace[]>("/api/workspaces")).data,
    enabled,
  });

// The selected workspace is where every chip on screen belongs, since the app only shows one workspace at a time.
export const useSelectedWorkspace = (): Workspace | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: workspaces } = useFetchWorkspaces();
  return workspaces?.find((w) => w.id === selectedWorkspaceId);
};

// F5 exception: repairs an empty or stale selection (or a slug a rename moved) so every workspace-scoped query works even where no switcher renders (onboarding). getState() avoids clobbering a just-created id before refetch.
export const useEnsureWorkspaceSelected = (enabled: boolean) => {
  const { data: workspaces } = useFetchWorkspaces(enabled);
  useEffect(() => {
    if (!workspaces) return;
    const { selectedWorkspaceId, selectedWorkspaceSlug, selectWorkspace } = useWorkspaceStore.getState();
    const selected = workspaces.find((w) => w.id === selectedWorkspaceId) ?? workspaces[0];
    if (selected?.id !== selectedWorkspaceId || selected?.slug !== selectedWorkspaceSlug) {
      selectWorkspace(selected?.id ?? "", selected?.slug ?? "");
    }
  }, [workspaces]);
};

const getMyRoleKey = "getMyRole";

// Mirrors internal/tenancy/handler.go's meResponse; keep the two in sync.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
}

// Shared with the workspace switcher, which reads the target workspace's role before it navigates.
export const myRoleQuery = (workspaceId: string) =>
  queryOptions({
    queryKey: [getMyRoleKey, workspaceId],
    queryFn: async () => (await api.get<MyWorkspaceInfo>(`/api/workspaces/${workspaceId}/me`)).data,
    enabled: workspaceId !== "",
  });

// Keyed by workspaceId so switching workspaces refetches; multiple call sites share the cache via dedupe.
export const useFetchMyRole = (workspaceId: string) => useQuery(myRoleQuery(workspaceId));

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
    mutationFn: async ({ id, ...body }: { id: string; name: string; slug?: string }) =>
      (await api.patch<Workspace>(`/api/workspaces/${id}`, body)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacesKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Gated server-side on workspaces:write in that workspace; the list read stays open to every member since chips need it.
export const useUpdateMentionChipTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { id: string; template: string }) =>
      (
        await api.patch<Workspace>(`/api/workspaces/${payload.id}/mention-chip-template`, {
          mention_chip_template: payload.template,
        })
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacesKey] });
      toast.success("Mention chip layout updated");
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
