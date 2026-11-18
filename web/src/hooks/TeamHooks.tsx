import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getProjectPeopleKey, getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getTeamKey, heldRoles, type Team, type TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";
import { followEach, refetchHolding, type LiveFollower } from "@/lib/live";

// The server scopes it: everything for an accounts:read holder, else only the workspaces the viewer manages.
export const useFetchTeam = (enabled = true) =>
  useQuery({
    queryKey: [getTeamKey],
    queryFn: async () => (await api.get<Team>("/api/team")).data,
    enabled,
    retry: false,
  });

const invalidateTeam = (client: QueryClient) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getTeamKey] }),
    client.invalidateQueries({ queryKey: [getWorkspacePeopleKey] }),
    client.invalidateQueries({ queryKey: [getProjectPeopleKey] }),
  ]);

const useTeamMutation = <TInput,>(request: (input: TInput) => Promise<unknown>, success: (input: TInput) => string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: request,
    onSuccess: async (_data, input) => {
      await invalidateTeam(client);
      toast.success(success(input));
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const memberPath = (workspaceId: string, userId: string) =>
  `/api/workspaces/${encodeURIComponent(workspaceId)}/members/${encodeURIComponent(userId)}`;

interface MemberChange {
  userId: string;
  workspaceId: string;
  // A person's whole set of roles in the workspace; left out, the change takes them out of it.
  roleIds?: string[];
  // Whether they are in the workspace yet, so giving a role adds them.
  joining?: boolean;
  done: string;
  failed: string;
}

const useChangeMember = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("members:write", ({ userId, workspaceId, roleIds, joining }: MemberChange) => {
      const path = memberPath(workspaceId, userId);
      if (!roleIds) return api.delete(path);
      if (joining) return api.put(path, { role_ids: roleIds });
      return api.patch(path, { role_ids: roleIds });
    }),
    onSuccess: async (_data, change) => {
      await invalidateTeam(client);
      toast.success(change.done);
    },
    onError: (error, change) => toast.error(`${change.failed}: ${errorMessage(error)}`),
  });
};

// Giving, taking and removing from a role on the Access page; each saves at once and names who and what in its toast.
export const useRoleHolders = (workspaceId: string, workspaceName: string) => {
  const change = useChangeMember();
  const held = (person: TeamPerson) => {
    const membership = person.workspaces.find((candidate) => candidate.workspace_id === workspaceId);
    return membership && heldRoles(membership).map((role) => role.id);
  };
  const give = (person: TeamPerson, role: { id: string; name: string }) => {
    const roleIds = held(person);
    change.mutate({
      userId: person.id,
      workspaceId,
      roleIds: [...(roleIds ?? []), role.id],
      joining: !roleIds,
      done: roleIds ? `${role.name} given to ${personName(person)}` : `${personName(person)} added to ${workspaceName} as ${role.name}`,
      failed: `Couldn't give ${role.name} to ${personName(person)}`,
    });
  };
  const take = (person: TeamPerson, role: { id: string; name: string }) =>
    change.mutate({
      userId: person.id,
      workspaceId,
      roleIds: (held(person) ?? []).filter((id) => id !== role.id),
      done: `${role.name} taken from ${personName(person)}`,
      failed: `Couldn't take ${role.name} from ${personName(person)}`,
    });
  const remove = (person: TeamPerson) =>
    change.mutate({
      userId: person.id,
      workspaceId,
      done: `${personName(person)} removed from ${workspaceName}`,
      failed: `Couldn't remove ${personName(person)} from ${workspaceName}`,
    });
  return { give, take, remove, pending: change.isPending };
};

// The same words whatever the person had: who closes an account learns nothing about their computers.
const computersDisconnected = "Their computers were disconnected.";

export const useUpdateAccountStatus = () =>
  useTeamMutation(
    requires("accounts:write", ({ id, status }: { id: string; status: "active" | "disabled" }) => api.patch(`/api/auth/accounts/${encodeURIComponent(id)}`, { status })),
    ({ status }) => (status === "disabled" ? `Account disabled. ${computersDisconnected}` : "Account status updated"),
  );

export const useRemoveAccount = () =>
  useTeamMutation(requires("accounts:delete", (id: string) => api.delete(`/api/auth/accounts/${encodeURIComponent(id)}`)), () => `Account removed. ${computersDisconnected}`);

// The Team follows account and membership changes made anywhere; presence frames name nobody, so it refetches whole.
// A person's Welcome wizard progress refetches the Team only while it holds that person.
export const teamFollower: LiveFollower = {
  "welcome.progress_changed": ({ user_id: userID }: { user_id: string }, { client }) =>
    refetchHolding<Team>(client, [getTeamKey], (team) => team.people.some((person) => person.id === userID)),
  ...followEach(
    [
      "account.admitted",
      "account.disabled",
      "account.reactivated",
      "account.removed",
      "account.restored",
      "account.presence_changed",
      "account.profile_updated",
      "workspace.member.added",
      "workspace.member.removed",
      "workspace.member.updated",
      "access.grant.changed",
    ],
    (_payload: unknown, { client }) => client.invalidateQueries({ queryKey: [getTeamKey], exact: true }),
  ),
};
