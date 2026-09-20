import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type {
  Account,
  ActiveInvitation,
  CreateInvitationFormData,
  CreatedInvitation,
  InvitationAcceptance,
  InvitationPreview,
  InvitationProvider,
  RedeemedInvitation,
} from "@/models/Invitation";

export const getInvitationsKey = "getInvitations";
export const getAccountsKey = "getAccounts";

export const useFetchInvitations = () =>
  useQuery({
    queryKey: [getInvitationsKey],
    queryFn: async () => {
      const data = (await api.get<ActiveInvitation[] | { invitations: ActiveInvitation[] }>("/api/invitations")).data;
      return Array.isArray(data) ? data : data.invitations;
    },
  });

export const useCreateInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateInvitationFormData) =>
      (await api.post<CreatedInvitation>("/api/invitations", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInvitationsKey] });
      toast.success("Invitation created — copy the link now");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRevokeInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/invitations/${encodeURIComponent(id)}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInvitationsKey] });
      toast.success("Invitation revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchInvitationPreview = (token: string) =>
  useQuery({
    queryKey: ["getInvitationPreview", token],
    queryFn: async () => (await api.post<InvitationPreview>("/api/invitations/preview", { token })).data,
    enabled: token !== "",
    retry: false,
  });

export const useStartInvitationOAuth = () =>
  useMutation({
    mutationFn: async ({ provider, token }: { provider: InvitationProvider; token: string }) =>
      (await api.post<{ url: string }>("/api/invitations/oauth", { provider, token })).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useCreateInvitationAcceptance = () =>
  useMutation({
    mutationFn: async (input: { token?: string; acceptance_token?: string }) =>
      (await api.post<InvitationAcceptance>("/api/invitations/acceptance", input)).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useRedeemInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (acceptance_token: string) =>
      (await api.post<RedeemedInvitation>("/api/invitations/redeem", { acceptance_token })).data,
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getInvitationsKey] }),
        client.invalidateQueries({ queryKey: ["getWorkspaces"] }),
        client.invalidateQueries({ queryKey: ["getWorkspaceMembers"] }),
        client.invalidateQueries({ queryKey: ["getMe"] }),
      ]);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchAccounts = () =>
  useQuery({
    queryKey: [getAccountsKey],
    queryFn: async () => {
      const data = (await api.get<Account[] | { accounts: Account[] }>("/api/auth/accounts")).data;
      return Array.isArray(data) ? data : data.accounts;
    },
  });

export const useUpdateAccountStatus = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: "active" | "disabled" }) =>
      (await api.patch<Account>(`/api/auth/accounts/${encodeURIComponent(id)}`, { status })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAccountsKey] });
      toast.success("Account status updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveAccount = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/auth/accounts/${encodeURIComponent(id)}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAccountsKey] });
      toast.success("Account removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
