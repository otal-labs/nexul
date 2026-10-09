import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { referenceDataOptions } from "@/lib/queryClient";
import type { MentionName } from "@/models/Doc";
import { personLabel, unknownPerson, type PeopleList, type Person } from "@/models/Person";

export const getWorkspacePeopleKey = "getWorkspacePeople";

// Any member may read it, unlike the members list, which needs members:write.
const useWorkspacePeople = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getWorkspacePeopleKey, workspaceId],
    queryFn: () => api.get<PeopleList>(`/api/workspaces/${workspaceId}/people`),
    enabled: !!workspaceId,
    ...referenceDataOptions,
  });

// A person mention's live name: their display name, "unknown" once People lacks them, the saved login until it loads.
export const useMentionName = (workspaceId: string | undefined): MentionName => {
  const { data } = useWorkspacePeople(workspaceId);
  return (userId) => {
    if (!data) return undefined;
    const person = data.people.find((p) => p.user_id === userId);
    return person ? personLabel(person) : "unknown";
  };
};

export const usePersonLookup = (workspaceId: string | undefined) => {
  const { data } = useWorkspacePeople(workspaceId);
  // Chat holds user ids and tickets hold logins, so either resolves.
  return (key: string): Person => data?.people.find((p) => p.user_id === key || p.login === key) ?? unknownPerson(key);
};
