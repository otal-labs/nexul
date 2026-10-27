import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { SearchFound } from "@nexul/client-core/search";

import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { pause } from "@/lib/pause";

export const searchWorkspaceKey = "searchWorkspace";

// The command palette's search over every kind in the selected workspace, fetched once typing pauses for 150ms.
export const useSearchWorkspace = (text: string) => {
  const q = text.trim();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [searchWorkspaceKey, workspaceId, q],
    enabled: q.length >= 2 && !!workspaceId,
    placeholderData: keepPreviousData,
    queryFn: async ({ signal }) => {
      await pause(150, signal);
      const res = await api.get<SearchFound>("/api/search", { params: { q, workspace_id: workspaceId, limit: 5 }, signal });
      return res.data;
    },
  });
};
