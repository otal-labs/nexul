import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { MentionChipData, MentionRef, MentionSearchResult } from "@/models/Mention";

export const resolveMentionsKey = "resolveMentions";
export const searchMentionsKey = "searchMentions";

// One batch request for every ref in a render; a gone target is omitted, so the client falls back to the label. A ticket key resolves in the selected workspace.
export const resolveMentions = async (refs: MentionRef[]): Promise<MentionChipData[]> => {
  if (refs.length === 0) return [];
  const workspaceId = useWorkspaceStore.getState().selectedWorkspaceId;
  const res = await api.post<{ chips: MentionChipData[] }>("/api/mentions/resolve", { refs, ...(workspaceId && { workspace_id: workspaceId }) });
  return res.data.chips;
};

// Backs the @ picker's autocomplete: people of the selected workspace, tickets, and docs.
export const searchMentions = async (query: string, limit = 8): Promise<MentionSearchResult[]> => {
  const workspaceId = useWorkspaceStore.getState().selectedWorkspaceId;
  const res = await api.get<{ results: MentionSearchResult[] }>("/api/mentions/search", {
    params: { q: query, limit, ...(workspaceId && { workspace_id: workspaceId }) },
  });
  return res.data.results;
};

// Stable, order-independent cache key so every chip in the same document render shares one query.
export const refsKey = (refs: MentionRef[]): string =>
  refs
    .map((ref) => `${ref.type}:${ref.id}`)
    .sort()
    .join(",");

export const useResolveMentions = (refs: MentionRef[]) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [resolveMentionsKey, workspaceId, refsKey(refs)],
    queryFn: () => resolveMentions(refs),
    enabled: refs.length > 0,
  });
};
