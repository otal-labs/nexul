import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type {
  BootstrapResponse,
  BootstrapStatus,
  ConnectionToken,
  InstanceSettings,
  LoginMatch,
  MeResponse,
  MintPATResponse,
  OptionalProvider,
  PersonalAccessToken,
  User,
} from "@/models/User";

export const getMeKey = "getMe";
const getSettingsKey = "getSettings";
const getMembersKey = "getMembers";
const lookupMembersKey = "lookupMembers";
const getPATsKey = "getPATs";
const getBootstrapStatusKey = "getBootstrapStatus";

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
      await client.invalidateQueries({ queryKey: [getMeKey], refetchType: "all" });
      toast.success("Profile updated");
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
      await client.invalidateQueries({ queryKey: [getSettingsKey] });
      toast.success("Instance URL updated — connection tokens regenerate on next issue");
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

// Gated server-side on workspaces:write; the read above stays open to any user since chips need it.
export const useUpdateMentionChipTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (mention_chip_template: string) =>
      (
        await api.patch<{ mention_chip_template: string }>("/api/auth/settings/mention-chip-template", {
          mention_chip_template,
        })
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getSettingsKey] });
      toast.success("Mention chip layout updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useGenerateConnectionToken = () =>
  useMutation({
    mutationFn: async () => (await api.post<ConnectionToken>("/api/auth/connection-token")).data,
    onSuccess: () => toast.success("Connection token generated — copy it now, it won't be shown again"),
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
      toast.success("Token created — copy it now, it won't be shown again");
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

export const useFetchMembers = () =>
  useQuery({
    queryKey: [getMembersKey],
    queryFn: async () => (await api.get<{ members: string[] }>("/api/auth/members")).data,
  });

export const useAddMember = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (login: string) =>
      (await api.post<{ members: string[] }>("/api/auth/members", { login })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMembersKey] });
      toast.success("Member added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveMember = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (login: string) =>
      (await api.delete<{ members: string[] }>(`/api/auth/members/${encodeURIComponent(login)}`)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getMembersKey] });
      toast.success("Member removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const LOOKUP_DEBOUNCE_MS = 300;

// Rejects with the cancel reason if aborted first, so the fetch never fires.
const sleepUnlessAborted = (ms: number, signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(signal.reason);
      },
      { once: true },
    );
  });

// Debounced via the queryFn's own sleep (F5), aborted by `signal` on each keystroke; disabled for emails (GitHub-only).
export const useLookupMembers = (q: string) =>
  useQuery({
    queryKey: [lookupMembersKey, q],
    queryFn: async ({ signal }) => {
      await sleepUnlessAborted(LOOKUP_DEBOUNCE_MS, signal);
      return (await api.get<{ matches: LoginMatch[] }>("/api/auth/members/lookup", { params: { q }, signal })).data;
    },
    enabled: q.trim().length >= 2 && !q.includes("@"),
    staleTime: 60_000,
    placeholderData: keepPreviousData,
  });

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
