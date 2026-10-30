import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getInvitationPreviewKey, invitationRefusal } from "@/models/Invitation";
import type {
  ActiveInvitation,
  CreateInvitationFormData,
  CreatedInvitation,
  InvitationAcceptance,
  InvitationList,
  InvitationPreview,
  InvitationProvider,
  RedeemedInvitation,
} from "@/models/Invitation";

export const getInvitationsKey = "getInvitations";

export const useFetchInvitations = () =>
  useQuery({
    queryKey: [getInvitationsKey],
    queryFn: async (): Promise<InvitationList> => {
      const data = (await api.get<ActiveInvitation[] | InvitationList>("/api/invitations")).data;
      return Array.isArray(data) ? { invitations: data, accepted: [], ended: [] } : data;
    },
  });

export const useCreateInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("members:write", async (input: CreateInvitationFormData) =>
      (await api.post<CreatedInvitation>("/api/invitations", input)).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInvitationsKey] });
      toast.success("Invitation created. Copy the link now.");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRevokeInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("members:write", async (id: string) => api.delete(`/api/invitations/${encodeURIComponent(id)}`)),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInvitationsKey] });
      toast.success("Invitation revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchInvitationPreview = (token: string) =>
  useQuery({
    queryKey: [getInvitationPreviewKey, token],
    queryFn: requires("none", async () => (await api.post<InvitationPreview>("/api/invitations/preview", { token })).data),
    enabled: token !== "",
    retry: false,
  });

export const useStartInvitationOAuth = () =>
  useMutation({
    // An acceptance instead of the link starts another provider after a round trip ("Not you?").
    mutationFn: requires("none", async ({ provider, token, acceptanceToken }: { provider: InvitationProvider; token?: string; acceptanceToken?: string }) =>
      (await api.post<{ url: string }>("/api/invitations/oauth", acceptanceToken ? { provider, acceptance_token: acceptanceToken } : { provider, token })).data),
    onError: (error) => toast.error(errorMessage(error)),
  });

// A refusal is the invite page's own screen, so only other failures toast.
const toastUnlessRefused = (error: unknown) => {
  if (invitationRefusal(error)) return;
  toast.error(errorMessage(error));
};

export const useCreateInvitationAcceptance = () =>
  useMutation({
    mutationFn: requires("none", async (input: { token?: string; acceptance_token?: string }) =>
      (await api.post<InvitationAcceptance>("/api/invitations/acceptance", input)).data),
    onError: toastUnlessRefused,
  });

export const useRedeemInvitation = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (acceptance_token: string) =>
      (await api.post<RedeemedInvitation>("/api/invitations/redeem", { acceptance_token })).data),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getInvitationsKey] }),
        client.invalidateQueries({ queryKey: ["getWorkspaces"] }),
        client.invalidateQueries({ queryKey: ["getWorkspacePeople"] }),
        client.invalidateQueries({ queryKey: ["getMe"] }),
      ]);
    },
    onError: toastUnlessRefused,
  });
};
