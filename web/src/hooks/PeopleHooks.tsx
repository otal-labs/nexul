import { useQuery, type QueryClient } from "@tanstack/react-query";

import { unknownPerson, type PeopleList, type Person } from "@nexul/client-core/person";

import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followEach, refetchHolding, type LiveFollower } from "@/lib/live";

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

// Every people list that shows the person, in any workspace or project.
const refetchShowing = (client: QueryClient, userId: string) =>
  Promise.all(
    [getWorkspacePeopleKey, getProjectPeopleKey].map((key) => refetchHolding(client, [key], (people: Person[]) => people.some((p) => p.user_id === userId))),
  );

interface MemberPayload {
  user_id: string;
  workspace_id: string;
  // Set on workspace.member.updated: the workspace's projects, whose access the change can move.
  project_ids?: string[];
}

export const peopleFollower: LiveFollower = {
  // A new name or picture reaches every open screen that shows the person.
  ...followEach(["account.profile_updated", "account.removed"], ({ account_id }: { account_id: string }, { client }) => refetchShowing(client, account_id)),
  "workspace.member.added": ({ workspace_id }: MemberPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getWorkspacePeopleKey, workspace_id], exact: true }),
  "workspace.member.removed": ({ user_id }: MemberPayload, { client }) => refetchShowing(client, user_id),
  // A role or Restricted change can open or close projects for the member; the pickers offer who may open each.
  "workspace.member.updated": ({ project_ids }: MemberPayload, { client }) =>
    Promise.all((project_ids ?? []).map((id) => client.invalidateQueries({ queryKey: [getProjectPeopleKey, id], exact: true }))),
  "access.grant.changed": ({ resource_type, resource_id }: { resource_type: string; resource_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: resource_type === "project" ? [getProjectPeopleKey, resource_id] : [getProjectPeopleKey] }),
};
