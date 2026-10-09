import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { REPOSITORY_SEARCH_MIN_LENGTH, type Installation, type Repo, type ScanResult } from "@/models/Repository";
import { pause } from "@/lib/pause";

const getRepositoriesKey = "repositories";
const getInstallationsKey = "repository-installations";

const searchDebounceMs = 250;

// A refetch of a loaded search (e.g. focus after installing the App) skips the pause and bypasses the server cache.
export const useSearchRepositories = (text: string) => {
  const q = text.trim();
  return useQuery({
    queryKey: [getRepositoriesKey, q],
    enabled: q.length >= REPOSITORY_SEARCH_MIN_LENGTH,
    staleTime: 30_000,
    refetchOnWindowFocus: "always",
    placeholderData: keepPreviousData,
    queryFn: async ({ client, queryKey, signal }) => {
      const refetching = client.getQueryData(queryKey) !== undefined;
      if (!refetching) await pause(searchDebounceMs, signal);
      const params = refetching ? { q, refresh: 1 } : { q };
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

// No onSuccess/onError toasting: the repository step renders the scan's loading/error/empty states inline
// rather than as a toast, since "not installed" and "nothing found" are real UI states, not failures to dismiss.
export const useScanRepository = () =>
  useMutation({
    mutationFn: async (input: { owner: string; name: string; ref?: string }) =>
      (await api.post<ScanResult>("/api/repositories/scan", input)).data,
  });
