import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { MembersList } from "@/models/Member";

const getWorkspaceMembersKey = "getWorkspaceMembers";

// 403s for a viewer without members:write, so direct navigation gets the server's real answer.
export const useFetchWorkspaceMembers = (workspaceId: string) =>
  useQuery({
    queryKey: [getWorkspaceMembersKey, workspaceId],
    queryFn: async () => (await api.get<MembersList>(`/api/workspaces/${workspaceId}/members`)).data,
    enabled: workspaceId !== "",
  });

export const useInviteWorkspaceMember = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ login, roleId }: { login: string; roleId: string }) =>
      api.post(`/api/workspaces/${workspaceId}/members`, { login, role_id: roleId }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspaceMembersKey, workspaceId] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveWorkspaceMember = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) =>
      api.delete(`/api/workspaces/${workspaceId}/members/${encodeURIComponent(userId)}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspaceMembersKey, workspaceId] });
      toast.success("Member removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCancelWorkspaceInvite = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (login: string) =>
      api.delete(`/api/workspaces/${workspaceId}/invites/${encodeURIComponent(login)}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspaceMembersKey, workspaceId] });
      toast.success("Invite withdrawn");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useChangeWorkspaceMemberRole = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ userId, roleId }: { userId: string; roleId: string }) =>
      api.patch(`/api/workspaces/${workspaceId}/members/${encodeURIComponent(userId)}`, { role_id: roleId }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspaceMembersKey, workspaceId] });
      toast.success("Role updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
