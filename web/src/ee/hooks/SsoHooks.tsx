import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { SsoFormData, SsoPublic, SsoSettings } from "@/ee/models/Sso";
import type { LiveFollower } from "@/lib/live";

export const getSsoKey = "ssoSettings";
export const getSsoPublicKey = "ssoPublic";

export const useSsoSettings = () =>
  useQuery({
    queryKey: [getSsoKey],
    queryFn: async () => (await api.get<SsoSettings>("/api/instance/sso")).data,
    staleTime: 5 * 60 * 1000,
  });

// The sign-in page reads this signed out, so it stays fresh for a minute and no longer.
export const usePublicSso = () =>
  useQuery({
    queryKey: [getSsoPublicKey],
    queryFn: async () => (await api.get<SsoPublic>("/api/auth/sso")).data,
    staleTime: 60 * 1000,
  });

const keep = (client: QueryClient, data: SsoSettings) => {
  client.setQueryData([getSsoKey], data);
  return client.invalidateQueries({ queryKey: [getSsoPublicKey] });
};

// The refused reason (a provider that does not answer, a role nobody may join as) comes back for the form to show.
export const useSaveSso = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async (data: SsoFormData) => (await api.put<SsoSettings>("/api/instance/sso", data)).data),
    onSuccess: (data) => keep(client, data),
  });
};

// Turning it on is refused until an instance Owner holds a passkey, so the reason shows as a toast.
export const useSetSsoOnly = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async (only: boolean) => (await api.put<SsoSettings>("/api/instance/sso", { only })).data),
    onSuccess: (data) => keep(client, data),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveSso = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.delete<SsoSettings>("/api/instance/sso")).data),
    onSuccess: (data) => {
      toast.success("Single sign-on removed");
      return keep(client, data);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// One request per check, so the ticker can say which part of the form is wrong.
export const useVerifySsoCheck = () =>
  requires("instance:write", (check: string, data: Record<string, string>) =>
    api.post<{ detail?: string } | undefined>("/api/instance/sso/verify", data, { params: { check } }).then((r) => r.data?.detail),
  );

export const ssoFollower: LiveFollower = {
  "instance.sso_changed": (_payload: unknown, { client }) =>
    Promise.all([client.invalidateQueries({ queryKey: [getSsoKey] }), client.invalidateQueries({ queryKey: [getSsoPublicKey] })]),
};
