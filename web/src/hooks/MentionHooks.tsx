import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { MentionChipData, MentionRef, MentionSearchResult } from "@/models/Mention";

export const resolveMentionsKey = "resolveMentions";
export const searchMentionsKey = "searchMentions";

// One batch request for every ref in a render; a gone target is omitted, so the client falls back to the label.
export const resolveMentions = async (refs: MentionRef[]): Promise<MentionChipData[]> => {
  if (refs.length === 0) return [];
  const res = await api.post<{ chips: MentionChipData[] }>("/api/mentions/resolve", { refs });
  return res.data.chips;
};

// Backs the @ picker's autocomplete (tickets + docs).
export const searchMentions = async (query: string, limit = 8): Promise<MentionSearchResult[]> => {
  const res = await api.get<{ results: MentionSearchResult[] }>("/api/mentions/search", {
    params: { q: query, limit },
  });
  return res.data.results;
};

// Stable, order-independent cache key so every chip in the same document render shares one query.
export const refsKey = (refs: MentionRef[]): string =>
  refs
    .map((ref) => `${ref.type}:${ref.id}`)
    .sort()
    .join(",");

export const useResolveMentions = (refs: MentionRef[]) =>
  useQuery({
    queryKey: [resolveMentionsKey, refsKey(refs)],
    queryFn: () => resolveMentions(refs),
    enabled: refs.length > 0,
  });
