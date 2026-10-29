import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { History, HistoryVersion } from "@/models/History";

export const getHistoryKey = "getHistory";
export const getHistoryVersionKey = "getHistoryVersion";

// Read when the dialog opens and never followed live: a list that moved under the reader would move the row they picked.
export const useFetchHistory = (historyUrl: string) =>
  useQuery({
    queryKey: [getHistoryKey, historyUrl],
    queryFn: async () => (await api.get<History>(historyUrl)).data,
    enabled: !!historyUrl,
    refetchOnMount: "always",
  });

// A version never changes once written, so the cache keeps it for as long as the dialog may come back to it.
export const useFetchHistoryVersion = (versionUrl: string | undefined) =>
  useQuery({
    queryKey: [getHistoryVersionKey, versionUrl],
    queryFn: async () => (await api.get<HistoryVersion>(versionUrl ?? "")).data,
    enabled: !!versionUrl,
    staleTime: Infinity,
  });
