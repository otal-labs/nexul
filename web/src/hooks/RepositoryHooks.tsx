import { useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Repo, ScanResult } from "@/models/Repository";

const getRepositoriesKey = "repositories";

// Installation repositories only — every repo this list returns already has the App installed, so a scan
// failing "not installed" on one of them is a race (App uninstalled since the list loaded), not the common case.
export const useFetchRepositories = () =>
  useQuery({
    queryKey: [getRepositoriesKey],
    queryFn: async () => (await api.get<{ repositories: Repo[] }>("/api/repositories")).data.repositories,
  });

// No onSuccess/onError toasting: the repository step renders the scan's loading/error/empty states inline
// rather than as a toast, since "not installed" and "nothing found" are real UI states, not failures to dismiss.
export const useScanRepository = () =>
  useMutation({
    mutationFn: async (input: { owner: string; name: string; ref?: string }) =>
      (await api.post<ScanResult>("/api/repositories/scan", input)).data,
  });
