import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { docChanged } from "@/hooks/DocHooks";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import type { Doc } from "@/models/Doc";

export const docHistoryUrl = (docId: string) => `/api/docs/${docId}/history`;
export const docVersionUrl = (docId: string, version: number) => `/api/docs/${docId}/versions/${version}`;

// The old text lands as a new version on top; the doc's live session picks the restored text up from the server.
export const useRestoreDocVersion = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("docs:write", async (version: number) => (await api.post<Doc>(`/api/docs/${docId}/versions/${version}/restore`)).data),
    onSuccess: async (doc) => {
      docChanged(client, doc);
      await client.invalidateQueries({ queryKey: [getHistoryKey, docHistoryUrl(docId)] });
      toast.success("Version restored");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
