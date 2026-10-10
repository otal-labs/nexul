import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { REPOSITORY_SEARCH_MIN_LENGTH, type Installation, type Repo, type ScanResult } from "@/models/Repository";
import { followEach, type LiveFollower } from "@/lib/live";
import { pause } from "@/lib/pause";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getRepositoriesKey = "repositories";
export const getInstallationsKey = "repository-installations";
const getInstallURLKey = "repository-install-url";

const searchDebounceMs = 250;

// A refetch of a loaded search (e.g. focus after installing the App) skips the pause and bypasses the server cache.
export const useSearchRepositories = (text: string) => {
  const q = text.trim();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getRepositoriesKey, workspaceId, q],
    enabled: q.length >= REPOSITORY_SEARCH_MIN_LENGTH,
    staleTime: 30_000,
    refetchOnWindowFocus: "always",
    placeholderData: keepPreviousData,
    queryFn: async ({ client, queryKey, signal }) => {
      const refetching = client.getQueryData(queryKey) !== undefined;
      if (!refetching) await pause(searchDebounceMs, signal);
      const params = refetching ? { workspace_id: workspaceId, q, refresh: 1 } : { workspace_id: workspaceId, q };
      return (await api.get<{ repositories: Repo[] }>("/api/repositories", { params, signal })).data.repositories;
    },
  });
};

// staleTime 0 for the same reason as useSearchRepositories: coming back from GitHub's install page shows the new account.
export const useFetchInstallations = () =>
  useQuery({
    queryKey: [getInstallationsKey],
    staleTime: 0,
    queryFn: async () =>
      (await api.get<{ installations: Installation[] }>("/api/repositories/installations")).data.installations,
  });

// GitHub's install page carrying the workspace, so an installation made from it lists here (signed by the server).
export const useFetchInstallURL = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getInstallURLKey, workspaceId],
    enabled: !!workspaceId,
    queryFn: async () =>
      (await api.get<{ url: string }>("/api/repositories/install-url", { params: { workspace_id: workspaceId } })).data.url,
  });
};

const installationWorkspacePath = (account: string, workspaceId: string) =>
  `/api/repositories/installations/${encodeURIComponent(account)}/workspaces/${encodeURIComponent(workspaceId)}`;

export const useAssignInstallation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ account, workspaceId }: { account: string; workspaceId: string }) => {
      await api.put(installationWorkspacePath(account, workspaceId));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [getInstallationsKey] });
      void queryClient.invalidateQueries({ queryKey: [getRepositoriesKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUnassignInstallation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ account, workspaceId }: { account: string; workspaceId: string }) => {
      await api.delete(installationWorkspacePath(account, workspaceId));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [getInstallationsKey] });
      void queryClient.invalidateQueries({ queryKey: [getRepositoriesKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface InstallationPayload {
  workspace_id: string;
}

// An assignment changes the installations list and the one workspace's repository lists; nothing else refetches.
export const repositoryFollower: LiveFollower = followEach(
  ["repository.installation.assigned", "repository.installation.unassigned"],
  ({ workspace_id }: InstallationPayload, { client }) =>
    Promise.all([
      client.invalidateQueries({ queryKey: [getInstallationsKey] }),
      client.invalidateQueries({ queryKey: [getRepositoriesKey, workspace_id] }),
    ]),
);

// No onSuccess/onError toasting: the repository step renders the scan's loading/error/empty states inline
// rather than as a toast, since "not installed" and "nothing found" are real UI states, not failures to dismiss.
export const useScanRepository = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useMutation({
    mutationFn: async (input: { owner: string; name: string; ref?: string }) =>
      (await api.post<ScanResult>("/api/repositories/scan", { ...input, workspace_id: workspaceId })).data,
  });
};
