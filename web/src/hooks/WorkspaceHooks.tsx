import { useEffect } from "react";
import { queryOptions, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate, type Location, type NavigateFunction } from "react-router";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { hasPermission, projectPermissions, type MyWorkspaceInfo } from "@/models/Permission";
import { replaceWorkspaceSlug, type Workspace, type WorkspaceUpdate } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { LiveFollower } from "@/lib/live";

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

export const getMyRoleKey = "getMyRole";

// Shared with the workspace switcher, which reads the target workspace's role before it navigates.
export const myRoleQuery = (workspaceId: string) =>
  queryOptions({
    queryKey: [getMyRoleKey, workspaceId],
    queryFn: async () => (await api.get<MyWorkspaceInfo>(`/api/workspaces/${workspaceId}/me`)).data,
    enabled: workspaceId !== "",
  });

// Keyed by workspaceId so switching workspaces refetches; multiple call sites share the cache via dedupe.
export const useFetchMyRole = (workspaceId: string) => useQuery(myRoleQuery(workspaceId));

// The frontend's single hasPermission(value) helper; no call site should compute permissions itself. A project
// action answers for the project in view, which is what the server checks for a Restricted member.
export const useHasPermission = (value: string): boolean => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const { data } = useFetchMyRole(selectedWorkspaceId);
  return hasPermission(projectPermissions(data, selectedProjectId), value);
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

// Applies a workspace's new name and slug everywhere at once: the list the switchers read, the selection, and the
// address bar when it sits inside that workspace, so nobody lands on a page the old slug no longer names.
export const followWorkspaceUpdate = (
  client: QueryClient,
  navigate: NavigateFunction,
  location: Pick<Location, "pathname" | "search" | "hash">,
  update: WorkspaceUpdate,
) => {
  client.setQueryData<Workspace[]>([getWorkspacesKey], (list) =>
    list?.map((w) => (w.id === update.workspace_id ? { ...w, name: update.name, slug: update.slug } : w)),
  );
  const { selectedWorkspaceId, selectedWorkspaceSlug, selectWorkspace } = useWorkspaceStore.getState();
  if (selectedWorkspaceId === update.workspace_id && selectedWorkspaceSlug !== update.slug) {
    const path = replaceWorkspaceSlug(location.pathname, selectedWorkspaceSlug, update.slug);
    selectWorkspace(update.workspace_id, update.slug);
    if (path) void navigate(`${path}${location.search}${location.hash}`, { replace: true });
  }
  void client.invalidateQueries({ queryKey: [getWorkspacesKey] });
};

// The Configuration General save: sends only the fields that changed. A refused slug shows inline on the form, so
// the caller handles the error rather than a toast here.
export const useUpdateWorkspace = () => {
  const client = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  return useMutation({
    mutationFn: async ({ id, ...body }: { id: string; name?: string; slug?: string }) =>
      (await api.patch<Workspace>(`/api/workspaces/${id}`, body)).data,
    onSuccess: (saved) =>
      followWorkspaceUpdate(client, navigate, location, { workspace_id: saved.id, name: saved.name, slug: saved.slug }),
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

export const workspaceFollower: LiveFollower = {
  "workspace.updated": (update: WorkspaceUpdate, { client, navigate, location }) => followWorkspaceUpdate(client, navigate, location, update),
};
