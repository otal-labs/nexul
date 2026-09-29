import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { unknownPerson, type PeopleList, type Person } from "@/models/Person";

export const getWorkspacePeopleKey = "getWorkspacePeople";

// Any member may read it, unlike the members list, which needs members:write.
export const usePersonLookup = (workspaceId: string | undefined) => {
  const { data } = useQuery({
    queryKey: [getWorkspacePeopleKey, workspaceId],
    queryFn: () => api.get<PeopleList>(`/api/workspaces/${workspaceId}/people`),
    enabled: !!workspaceId,
  });
  // Chat holds user ids and tickets hold logins, so either resolves.
  return (key: string): Person => data?.people.find((p) => p.user_id === key || p.login === key) ?? unknownPerson(key);
};
