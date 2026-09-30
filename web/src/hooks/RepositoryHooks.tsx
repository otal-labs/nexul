import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { REPOSITORY_SEARCH_MIN_LENGTH, type Installation, type Repo, type ScanResult } from "@/models/Repository";

const getRepositoriesKey = "repositories";
const getInstallationsKey = "repository-installations";

const searchDebounceMs = 250;

// Resolves after ms, or rejects when the query is cancelled: the key changing (another keystroke) aborts the
// signal, so only the last pause in typing goes on to fetch.
const pause = (ms: number, signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(signal.reason);
      },
      { once: true },
    );
  });

// Installation repositories matching the typed text, and nothing until it is long enough — every repo this returns
// already has the App installed, so a scan failing "not installed" on one of them is a race, not the common case.
// The debounce lives in the queryFn: a new key cancels the previous one during its pause, so no state or effect is
// needed. Refetching a key that already has data (window focus after GitHub's install page, or a stale one revisited)
// skips the pause and asks the server past its cache, so a newly installed account shows up. Data stays fresh for
// 30s so backspacing over recent searches costs no request.
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
