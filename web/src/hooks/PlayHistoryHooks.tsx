import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import { getApplicablePlaysKey, getWorkspacePlaysKey } from "@/hooks/PlayHooks";
import type { Play } from "@/models/Play";

export const playHistoryUrl = (playId: string) => `/api/plays/${playId}/history`;
export const playVersionUrl = (playId: string, version: number) => `/api/plays/${playId}/versions/${version}`;

// The old instructions land as a new version on top; the plays lists and the history list read again.
export const useRestorePlayVersion = (playId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:write", async (version: number) => (await api.post<Play>(`/api/plays/${playId}/versions/${version}/restore`)).data),
    onSuccess: async () => {
      await Promise.all([getWorkspacePlaysKey, getApplicablePlaysKey, getHistoryKey].map((key) => client.invalidateQueries({ queryKey: [key] })));
      toast.success("Version restored");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
