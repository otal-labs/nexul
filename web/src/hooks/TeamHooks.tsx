import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getProjectPeopleKey, getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getProjectAccessKey } from "@/hooks/ProjectHooks";
import type { MemberStep } from "@/models/MemberDraft";
import { getTeamKey, type Team } from "@/models/Team";
import { followEach, type LiveFollower } from "@/lib/live";

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
    client.invalidateQueries({ queryKey: [getProjectAccessKey] }),
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

// One of the Team dialog's held changes; the dialog reports a failure itself and names the change, so no toast here.
export const useApplyMemberStep = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, step }: { userId: string; step: MemberStep }) => {
      const path = memberPath(step.workspaceId, userId);
      if (step.kind === "add") return api.put(path, step.body);
      if (step.kind === "remove") return api.delete(path);
      return api.patch(path, step.body);
    },
    onSuccess: () => invalidateTeam(client),
  });
};

// The same words whatever the person had: who closes an account learns nothing about their computers.
const computersDisconnected = "Their computers were disconnected.";

export const useUpdateAccountStatus = () =>
  useTeamMutation(
    ({ id, status }: { id: string; status: "active" | "disabled" }) => api.patch(`/api/auth/accounts/${encodeURIComponent(id)}`, { status }),
    ({ status }) => (status === "disabled" ? `Account disabled. ${computersDisconnected}` : "Account status updated"),
  );

export const useRemoveAccount = () =>
  useTeamMutation((id: string) => api.delete(`/api/auth/accounts/${encodeURIComponent(id)}`), () => `Account removed. ${computersDisconnected}`);

// The Team follows account and membership changes made anywhere; presence frames name nobody, so it refetches whole.
export const teamFollower: LiveFollower = followEach(
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
);
