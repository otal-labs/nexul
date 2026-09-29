import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getTeamKey, type Team } from "@/models/Team";

// The server scopes it: everything for an instance administrator, else only the workspaces the viewer manages.
export const useFetchTeam = (enabled = true) =>
  useQuery({
    queryKey: [getTeamKey],
    queryFn: async () => (await api.get<Team>("/api/team")).data,
    enabled,
    retry: false,
  });

const useTeamMutation = <TInput,>(request: (input: TInput) => Promise<unknown>, success: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: request,
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getTeamKey] }),
        client.invalidateQueries({ queryKey: [getWorkspacePeopleKey] }),
      ]);
      toast.success(success);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface MemberTarget {
  workspaceId: string;
  userId: string;
}

const memberPath = ({ workspaceId, userId }: MemberTarget) =>
  `/api/workspaces/${encodeURIComponent(workspaceId)}/members/${encodeURIComponent(userId)}`;

export const useAddTeamMember = () =>
  useTeamMutation((input: MemberTarget & { roleId: string }) => api.put(memberPath(input), { role_id: input.roleId }), "Added to the workspace");

export const useChangeTeamMemberRole = () =>
  useTeamMutation((input: MemberTarget & { roleId: string }) => api.patch(memberPath(input), { role_id: input.roleId }), "Role updated");

export const useSetTeamMemberOverrides = () =>
  useTeamMutation(
    (input: MemberTarget & { allow: string[]; deny: string[] }) => api.patch(memberPath(input), { allow: input.allow, deny: input.deny }),
    "Overrides saved",
  );

export const useRemoveTeamMember = () =>
  useTeamMutation((input: MemberTarget) => api.delete(memberPath(input)), "Removed from the workspace");

export const useUpdateAccountStatus = () =>
  useTeamMutation(
    ({ id, status }: { id: string; status: "active" | "disabled" }) => api.patch(`/api/auth/accounts/${encodeURIComponent(id)}`, { status }),
    "Account status updated",
  );

export const useRemoveAccount = () =>
  useTeamMutation((id: string) => api.delete(`/api/auth/accounts/${encodeURIComponent(id)}`), "Account removed");
