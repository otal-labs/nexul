import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import { getMemoriesKey, getMemoryKey, memoryHistoryUrl } from "@/hooks/MemoryHooks";
import type { Memory } from "@/models/Memory";

// The old text lands as a new version on top, so the memory and its list of versions are read again.
export const useRestoreMemoryVersion = (memoryId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("memories:write", async (version: number) => (await api.post<Memory>(`/api/memories/${memoryId}/versions/${version}/restore`)).data),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getMemoriesKey] }),
        client.invalidateQueries({ queryKey: [getMemoryKey, memoryId] }),
        client.invalidateQueries({ queryKey: [getHistoryKey, memoryHistoryUrl(memoryId)] }),
      ]);
      toast.success("Version restored");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
