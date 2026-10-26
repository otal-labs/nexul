import { useQuery } from "@tanstack/react-query";

import { personLabel, unknownPerson, type PeopleList, type Person } from "@nexul/client-core/person";
import { findTicketPerson, type TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { MentionName } from "@/models/Doc";

export const getWorkspacePeopleKey = "getWorkspacePeople";

// Any member may read it, unlike the members list, which needs members:write.
const peopleQuery = defineQuery({
  key: getWorkspacePeopleKey,
  fetch: (workspaceId: string | undefined) => api.get<PeopleList>(`/api/workspaces/${workspaceId}/people`),
  refreshes: {
    "account.profile_updated": "all",
    "account.removed": "all",
    "workspace.member.added": { key: (p) => p.workspace_id },
    "workspace.member.removed": { key: (p) => p.workspace_id },
  },
  untilPushed: true,
});

const useWorkspacePeople = (workspaceId: string | undefined) =>
  useQuery({ ...peopleQuery.options(workspaceId), enabled: !!workspaceId });

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
  // Some records hold user ids and some logins, so either resolves.
  return (key: string): Person => data?.people.find((p) => p.user_id === key || p.login === key) ?? unknownPerson(key);
};

// A ticket's developer or tester in the workspace's directory.
export const useTicketPersonLookup = (workspaceId: string | undefined) => {
  const { data } = useWorkspacePeople(workspaceId);
  return (ref: TicketPersonRef): Person => findTicketPerson(data?.people, ref);
};
