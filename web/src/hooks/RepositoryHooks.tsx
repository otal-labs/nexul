import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Installation, Repo, ScanResult } from "@/models/Repository";

const getRepositoriesKey = "repositories";
const getInstallationsKey = "repository-installations";

// Installation repositories only — every repo this list returns already has the App installed, so a scan
// failing "not installed" on one of them is a race (App uninstalled since the list loaded), not the common case.
// staleTime 0 so returning from GitHub's install page refetches on window focus and the new account shows up.
export const useFetchRepositories = () =>
  useQuery({
    queryKey: [getRepositoriesKey],
    staleTime: 0,
    queryFn: async () => (await api.get<{ repositories: Repo[] }>("/api/repositories")).data.repositories,
  });

// staleTime 0 for the same reason as useFetchRepositories: coming back from GitHub's install page shows the new account.
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
