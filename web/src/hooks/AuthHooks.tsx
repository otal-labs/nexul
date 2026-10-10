import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";
import { useSessionStore } from "@/stores/sessionStore";
import { providerLabel } from "@/models/User";
import type {
  BootstrapResponse,
  BootstrapStatus,
  ConnectCode,
  ConnectionToken,
  Identity,
  InstanceSettings,
  MeResponse,
  MintPATResponse,
  OptionalProvider,
  PersonalAccessToken,
  Provider,
  Session,
  SessionClient,
  User,
} from "@/models/User";
import { followEach, type LiveFollower } from "@/lib/live";
import { getGitHubLinkKey } from "@/hooks/GitHubLinkHooks";

export const getMeKey = "getMe";
const getSettingsKey = "getSettings";
export const getPATsKey = "getPATs";
export const getSessionsKey = "getSessions";
export const getIdentitiesKey = "getIdentities";
export const getBootstrapStatusKey = "getBootstrapStatus";

export const useFetchMe = () =>
  useQuery({
    queryKey: [getMeKey],
    queryFn: async () => (await api.get<MeResponse>("/api/auth/me")).data,
    staleTime: 30_000,
  });

export const useCompleteOwnerWizard = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (instance_url: string) =>
      (await api.post<MeResponse>("/api/auth/onboarding/owner", { instance_url })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMeKey], refetchType: "all" });
      toast.success("Workspace ready");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCompleteFirstLogin = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => (await api.post<MeResponse>("/api/auth/onboarding/profile")).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMeKey], refetchType: "all" });
      toast.success("Welcome");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Sets the caller's optional name/avatar override; an empty string for either field clears back to provider-sourced.
export const useUpdateProfile = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { display_name: string; avatar_override_url: string }) =>
      (await api.put<User>("/api/auth/profile", payload)).data,
    onSuccess: async () => {
      // No toast: Settings answers in its Save button and the wizard moves on.
      await client.invalidateQueries({ queryKey: [getMeKey], refetchType: "all" });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchSettings = () =>
  useQuery({
    queryKey: [getSettingsKey],
    queryFn: async () => (await api.get<InstanceSettings>("/api/auth/settings")).data,
  });

export const useUpdateSettings = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (instance_url: string) =>
      (await api.put<InstanceSettings>("/api/auth/settings", { instance_url })).data,
    onSuccess: async () => {
      // No toast: the card's Save answers, and its row already says new connection tokens use the address.
      await client.invalidateQueries({ queryKey: [getSettingsKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The secret is write-only; the response never echoes it back.
export const useUpdateProviderOAuth = (provider: OptionalProvider, label: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (data: { client_id: string; client_secret: string }) =>
      (
        await api.put<{ provider: OptionalProvider; client_id: string; callback: string; configured: boolean }>(
          `/api/auth/settings/oauth/${provider}`,
          data,
        )
      ).data,
    onSuccess: async (data) => {
      await client.invalidateQueries({ queryKey: [getSettingsKey] });
      await client.invalidateQueries({ queryKey: [getBootstrapStatusKey] });
      toast.success(data.configured ? `${label} sign-in enabled` : `${label} sign-in disabled`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Mints and copies in one step; the caller shows the tick, so success needs no toast.
export const useCopyConnectionToken = () =>
  useMutation({
    mutationFn: async () => {
      const { token } = (await api.post<ConnectionToken>("/api/auth/connection-token")).data;
      await navigator.clipboard.writeText(token);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useListPATs = () =>
  useQuery({
    queryKey: [getPATsKey],
    queryFn: async () => (await api.get<{ tokens: PersonalAccessToken[] }>("/api/auth/tokens")).data,
  });

export const useMintPAT = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => (await api.post<MintPATResponse>("/api/auth/tokens", { name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getPATsKey] });
      toast.success("Token created. Copy it now. It won't be shown again.");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRevokePAT = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      (await api.delete<{ tokens: PersonalAccessToken[] }>(`/api/auth/tokens/${id}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getPATsKey] });
      toast.success("Token revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useListSessions = () =>
  useQuery({
    queryKey: [getSessionsKey],
    queryFn: async () => (await api.get<{ sessions: Session[] }>("/api/auth/sessions")).data,
  });

// A mutation, not a query: a code is issued only when the user asks, so visiting the page requests nothing.
export const useConnectCode = () =>
  useMutation({
    mutationFn: async () => (await api.post<ConnectCode>("/api/auth/connect-codes")).data,
  });

export const useSignOutSession = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/auth/sessions/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSessionsKey] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSignOutOtherSessions = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => api.delete("/api/auth/sessions/others"),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSessionsKey] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchIdentities = () =>
  useQuery({
    queryKey: [getIdentitiesKey],
    queryFn: async () => (await api.get<{ identities: Identity[] }>("/api/auth/identities")).data.identities,
  });

// The server answers the provider's authorize URL; the browser goes there itself and comes back to the Profile section.
export const useStartIdentityLink = () =>
  useMutation({
    mutationFn: async (provider: Provider) =>
      (await api.post<{ url: string }>("/api/auth/identities/link", { provider })).data,
    onSuccess: (data) => window.location.assign(data.url),
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useUnlinkIdentity = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (provider: Provider) => api.delete(`/api/auth/identities/${provider}`),
    onSuccess: async (_, provider) => {
      // Unlinking GitHub forgets its token too, so the GitHub row goes stale with the identities.
      await client.invalidateQueries({ queryKey: [getIdentitiesKey] });
      await client.invalidateQueries({ queryKey: [getGitHubLinkKey] });
      toast.success(`${providerLabel[provider]} unlinked`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Deletes the session server-side first; local state clears either way, since a failed delete still means leaving.
export const useLogout = () => {
  const navigate = useNavigate();
  const logout = useSessionStore((s) => s.logout);
  return useMutation({
    mutationFn: async () => api.delete("/api/auth/sessions/current"),
    onSettled: () => {
      logout();
      // Replace, not push: the current URL would 404 once the router rebuilds logged-out.
      navigate("/", { replace: true });
    },
  });
};

// Public and unauthenticated: decides whether to show the bootstrap page before any login is reachable.
export const useBootstrapStatus = () =>
  useQuery({
    queryKey: [getBootstrapStatusKey],
    queryFn: async () => (await api.get<BootstrapStatus>("/api/auth/bootstrap-status")).data,
  });

export const useBootstrap = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (data: { instance_url: string; client_id: string; client_secret: string; app_slug: string }) =>
      (await api.post<BootstrapResponse>("/api/auth/bootstrap", data)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getBootstrapStatusKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The metadata of a session that just signed in; never the token.
interface SessionCreatedPayload {
  session_id: string;
  user_id: string;
  client: SessionClient;
  platform: string;
  label: string;
}

const refetch = (client: QueryClient, key: string) => client.invalidateQueries({ queryKey: [key], exact: true });

export const authFollower: LiveFollower = {
  // The Devices list follows a phone connecting or a device being signed out; the viewer's own phone flips the page.
  "session.created": (p: SessionCreatedPayload, { client }) => {
    if (p.client === "phone" && p.user_id === client.getQueryData<MeResponse>([getMeKey])?.user.id) {
      useDeviceArrivalStore.getState().arrive({ id: p.session_id, platform: p.platform, label: p.label });
    }
    return refetch(client, getSessionsKey);
  },
  "session.revoked": (_payload: unknown, { client }) => refetch(client, getSessionsKey),
  ...followEach(["personal_access_token.minted", "personal_access_token.revoked"], (_payload: unknown, { client }) => refetch(client, getPATsKey)),
  // Someone else's new name or picture changes nothing in the viewer's own account.
  "account.profile_updated": ({ account_id }: { account_id: string }, { client }) =>
    account_id === client.getQueryData<MeResponse>([getMeKey])?.user.id && refetch(client, getMeKey),
};
