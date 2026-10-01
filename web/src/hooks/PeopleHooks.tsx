import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { unknownPerson, type PeopleList, type Person } from "@/models/Person";

export const getWorkspacePeopleKey = "getWorkspacePeople";
export const getProjectPeopleKey = "getProjectPeople";

// Any member may read it, unlike the members list, which needs members:write.
export const useFetchWorkspacePeople = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getWorkspacePeopleKey, workspaceId],
    queryFn: async () => (await api.get<PeopleList>(`/api/workspaces/${workspaceId}/people`)).data.people,
    enabled: !!workspaceId,
  });

// The people who may open a project, Restricted members without access left out: a ticket's developer and tester.
export const useFetchProjectPeople = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectPeopleKey, projectId],
    queryFn: async () => (await api.get<PeopleList>(`/api/projects/${projectId}/people`)).data.people,
    enabled: !!projectId,
  });

// One shared directory fetch; chat holds user ids and tickets hold logins, so either resolves.
export const usePersonLookup = (workspaceId: string | undefined) => {
  const { data: people } = useFetchWorkspacePeople(workspaceId);
  return (key: string): Person => people?.find((p) => p.user_id === key || p.login === key) ?? unknownPerson(key);
};

// For surfaces that carry no workspace id of their own (tickets, memories): they belong to the selected workspace.
export const usePerson = (key: string): Person => usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId))(key);
