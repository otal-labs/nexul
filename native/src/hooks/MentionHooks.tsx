import { useQuery } from "@tanstack/react-query";

import type { MentionChipData, MentionRef } from "@nexul/client-core/mention";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";

export const resolveMentionsKey = "resolveMentions";

// One batch for every doc and ticket a body mentions; a target that is gone is left out, and one the viewer can't read comes back locked.
const resolveMentionsQuery = defineQuery({
  key: resolveMentionsKey,
  fetch: requires("none", async (workspaceId: string | undefined, refs: MentionRef[]) =>
    (await api.post<{ chips: MentionChipData[] }>("/api/mentions/resolve", { refs, ...(workspaceId && { workspace_id: workspaceId }) })).chips),
  refreshes: { "access.grant.changed": "all" },
});

export const useResolveMentions = (workspaceId: string | undefined, refs: MentionRef[]) =>
  useQuery({ ...resolveMentionsQuery.options(workspaceId, refs), enabled: refs.length > 0 });
