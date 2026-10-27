import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import type { SearchFound } from "@nexul/client-core/search";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";

export const searchWorkspaceKey = "searchWorkspace";

// The workspace search the web's command palette reads, five title and five text hits per kind.
const searchWorkspaceQuery = defineQuery({
  key: searchWorkspaceKey,
  fetch: (workspaceId: string | undefined, q: string) =>
    api.get<SearchFound>(`/api/search?limit=5&workspace_id=${encodeURIComponent(workspaceId ?? "")}&q=${encodeURIComponent(q)}`),
  refreshes: { "access.grant.changed": "all" },
});

// Fetches once typing pauses for 250ms, so a word typed fast sends one request, not one per letter.
export const useSearchWorkspace = (workspaceId: string | undefined, text: string) => {
  const [q, setQ] = useState(text.trim());
  useEffect(() => {
    const timer = setTimeout(() => setQ(text.trim()), 250);
    return () => clearTimeout(timer);
  }, [text]);
  const query = useQuery({ ...searchWorkspaceQuery.options(workspaceId, q), enabled: !!workspaceId && q.length >= 2, placeholderData: keepPreviousData });
  return { ...query, settled: q === text.trim() };
};
