import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { createPasskey, getPasskey, passkeyCancelled } from "@/lib/passkeys";
import { useSessionStore } from "@/stores/sessionStore";
import type { Passkey, PasskeyCeremony, PasskeysResponse } from "@/models/Passkey";

export const getPasskeysKey = "getPasskeys";

const reportUnlessCancelled = (error: unknown) => {
  if (!passkeyCancelled(error)) toast.error(errorMessage(error));
};

export const useFetchPasskeys = () =>
  useQuery({
    queryKey: [getPasskeysKey],
    queryFn: async () => (await api.get<PasskeysResponse>("/api/auth/passkeys")).data,
  });

const startPasskey = requires("none", async () =>
  (await api.post<PasskeyCeremony<PublicKeyCredentialCreationOptionsJSON>>("/api/auth/passkeys/start")).data);

const savePasskey = requires("none", async (ceremony: string, credential: RegistrationResponseJSON | AuthenticationResponseJSON) =>
  (await api.post<Passkey>("/api/auth/passkeys", { ceremony, credential })).data);

// The browser makes the key between the two requests; the server only ever sees its public half.
export const useAddPasskey = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const start = await startPasskey();
      return savePasskey(start.ceremony, await createPasskey(start.options));
    },
    onSuccess: async (passkey) => {
      await client.invalidateQueries({ queryKey: [getPasskeysKey] });
      toast.success(`Passkey added: ${passkey.name}`);
    },
    onError: reportUnlessCancelled,
  });
};

export const useRenamePasskey = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async ({ id, name }: { id: string; name: string }) =>
      api.patch(`/api/auth/passkeys/${encodeURIComponent(id)}`, { name })),
    onSuccess: () => client.invalidateQueries({ queryKey: [getPasskeysKey] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeletePasskey = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (passkey: Passkey) => api.delete(`/api/auth/passkeys/${encodeURIComponent(passkey.id)}`)),
    onSuccess: async (_, passkey) => {
      await client.invalidateQueries({ queryKey: [getPasskeysKey] });
      toast.success(`${passkey.name} removed`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const startPasskeySignIn = requires("none", async () =>
  (await api.post<PasskeyCeremony<PublicKeyCredentialRequestOptionsJSON>>("/api/auth/passkeys/login/start")).data);

const finishPasskeySignIn = requires("none", async (ceremony: string, credential: RegistrationResponseJSON | AuthenticationResponseJSON) =>
  (await api.post<{ token: string }>("/api/auth/passkeys/login/finish", { ceremony, credential })).data);

export const usePasskeySignIn = () => {
  const login = useSessionStore((s) => s.login);
  const navigate = useNavigate();
  return useMutation({
    mutationFn: async () => {
      const start = await startPasskeySignIn();
      return finishPasskeySignIn(start.ceremony, await getPasskey(start.options));
    },
    onSuccess: (data) => {
      login(data.token);
      navigate("/", { replace: true });
    },
  });
};
