import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";
import { toast } from "sonner";

import { api, errorMessage, type ApiErrorBody } from "@/api/client";
import { getPATsKey } from "@/hooks/AuthHooks";
import {
  HARNESS_READINESS_COPY,
  PAIR_FIELDS,
  type Computer,
  type ComputerSetup,
  type CreateComputerTunnelFormData,
  type HarnessProject,
  type HarnessProvider,
  type HarnessReadiness,
  type MCPToken,
  type MintedMCPToken,
  type PairComputerFormData,
  type PairField,
  type PairingDefaults,
  type PairingDefaultsFormData,
  type ProjectLink,
  type ProjectLinkFormData,
  type SetupRun,
  type TunnelPrerequisite,
  type TunnelStatus,
} from "@/models/Pairing";

export const getComputersKey = "getComputers";
export const getPairingDefaultsKey = "getPairingDefaults";
export const getProjectLinkKey = "getProjectLink";
export const getHarnessProjectsKey = "getHarnessProjects";
export const getPairingPresenceKey = "getPairingPresence";
export const getHarnessProvidersKey = "getHarnessProviders";
export const getHarnessResolveKey = "getHarnessResolve";
export const getTunnelStatusKey = "getTunnelStatus";
export const getTunnelTokenKey = "getTunnelToken";
export const getMCPTokenKey = "getMCPToken";
export const getComputerSetupKey = "getComputerSetup";

// The four NotConfiguredReason values internal/pairing.ResolveTarget can fail with, mapped onto HarnessReadiness states.
const RESOLVE_REASON_TO_STATE: Record<string, Exclude<HarnessReadiness["state"], "ready" | "offline">> = {
  unpaired: "unpaired",
  expired_token: "expired",
  no_default: "no_harness_project",
  no_default_computer: "no_default_computer",
};

interface ResolveResponse {
  ok: boolean;
  reason?: string;
  computer_id?: string;
  provider?: string;
  model?: string;
}

export const useListComputers = () =>
  useQuery({
    queryKey: [getComputersKey],
    queryFn: async () => (await api.get<{ computers: Computer[] }>("/api/pairing/computers")).data.computers,
  });

interface PairComputerInput {
  // Set for a computer that already has a row (a computer tunnel): pairs at its own address.
  computerId?: string | undefined;
  form: PairComputerFormData;
}

// No error toast: the pairing form shows each failure on the field that caused it.
export const usePairComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ computerId, form }: PairComputerInput) => {
      if (computerId) return (await api.post<Computer>(`/api/pairing/computers/${computerId}/pair`, { token: form.token })).data;
      return (await api.post<Computer>("/api/pairing/computers", form)).data;
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success("Computer paired");
    },
  });
};

// The field each failure belongs to, from the error envelope's errors map; empty when the failure has no field.
export const pairFieldErrors = (error: unknown): [PairField, string][] => {
  const errors = (error as AxiosError<ApiErrorBody> | null)?.response?.data?.errors ?? {};
  return PAIR_FIELDS.flatMap((field): [PairField, string][] => {
    const message = errors[field]?.[0];
    return message ? [[field, message]] : [];
  });
};

// No error toast: a missing prerequisite renders as the step's own alert card, anything else inline under the form.
export const useCreateComputerTunnel = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateComputerTunnelFormData) =>
      (await api.post<Computer>("/api/pairing/computers/tunnel", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
    },
  });
};

// Read once; every later change arrives as a computer.tunnel_status_changed frame patched in by setCachedTunnelStatus.
export const useFetchTunnelStatus = (computerId: string) =>
  useQuery({
    queryKey: [getTunnelStatusKey, computerId],
    queryFn: async () => (await api.get<TunnelStatus>(`/api/pairing/computers/${computerId}/tunnel/status`)).data,
    enabled: !!computerId,
  });

export const useFetchTunnelToken = (computerId: string) =>
  useQuery({
    queryKey: [getTunnelTokenKey, computerId],
    queryFn: async () => (await api.get<{ token: string }>(`/api/pairing/computers/${computerId}/tunnel/token`)).data.token,
    enabled: !!computerId,
    staleTime: Infinity,
  });

// The tunnel routes add a reason to the error envelope when an instance prerequisite is missing.
export const tunnelPrerequisite = (error: unknown): TunnelPrerequisite | undefined => {
  const reason = (error as AxiosError<{ reason?: string }> | null)?.response?.data?.reason;
  if (reason === "cloudflare_not_connected" || reason === "zero_trust_disabled") return reason;
  return undefined;
};

export interface TunnelStatusChangedPayload extends TunnelStatus {
  computer_id: string;
}

export const setCachedTunnelStatus = (client: QueryClient, p: TunnelStatusChangedPayload) => {
  client.setQueryData<TunnelStatus>([getTunnelStatusKey, p.computer_id], {
    tunnel: p.tunnel,
    harness_reachable: p.harness_reachable,
    ...(p.harness_version && { harness_version: p.harness_version }),
  });
};

export const useRepairComputer = (id: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: PairComputerFormData) =>
      (await api.post<Computer>(`/api/pairing/computers/${id}/repair`, input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success("Computer re-paired");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/pairing/computers/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      await client.invalidateQueries({ queryKey: [getPATsKey] });
      toast.success("Computer removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Read once; setup turn and confirmation pushes invalidate it, so the row and the Set up step follow a run live.
export const useFetchComputerSetup = (computerId: string) =>
  useQuery({
    queryKey: [getComputerSetupKey, computerId],
    queryFn: async () => (await api.get<ComputerSetup>(`/api/pairing/computers/${computerId}/setup`)).data,
    enabled: !!computerId,
  });

// Without a provider it starts setup for every provider; with one it re-runs only that provider.
export const useRunSetup = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (provider?: string) => {
      const url = provider
        ? `/api/pairing/computers/${computerId}/setup/providers/${encodeURIComponent(provider)}/retry`
        : `/api/pairing/computers/${computerId}/setup/runs`;
      return (await api.post<SetupRun>(url)).data;
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputerSetupKey, computerId] });
      toast.success("Setup started");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Metadata only; the raw token exists solely on the mint mutation's data.
export const useFetchMCPToken = (computerId: string) =>
  useQuery({
    queryKey: [getMCPTokenKey, computerId],
    queryFn: async () =>
      (await api.get<{ mcp_token: MCPToken | null }>(`/api/pairing/computers/${computerId}/mcp-token`)).data.mcp_token,
  });

const invalidateMCPToken = async (client: QueryClient, computerId: string) => {
  await client.invalidateQueries({ queryKey: [getMCPTokenKey, computerId] });
  await client.invalidateQueries({ queryKey: [getPATsKey] });
};

export const useMintMCPToken = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () =>
      (await api.post<MintedMCPToken>(`/api/pairing/computers/${computerId}/mcp-token`)).data,
    onSuccess: async () => {
      await invalidateMCPToken(client, computerId);
      toast.success("MCP token created — copy it now, it won't be shown again");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRevokeMCPToken = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/pairing/computers/${computerId}/mcp-token`);
    },
    onSuccess: async () => {
      await invalidateMCPToken(client, computerId);
      toast.success("MCP token revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Polled: presence flips on browser connect/disconnect and T3 restarts, not on any user action here.
export const useFetchPresence = () =>
  useQuery({
    queryKey: [getPairingPresenceKey],
    queryFn: async () => (await api.get<{ computers: Record<string, string> }>("/api/pairing/presence")).data.computers,
    refetchInterval: 15_000,
  });

// Joins the read-only resolve route with presence: ready needs both a resolved computer and a connected session.
// Returns undefined while either query is loading, so callers never flash "offline" before presence has an answer.
export const useHarnessReadiness = (projectId?: string): HarnessReadiness | undefined => {
  const resolve = useQuery({
    queryKey: [getHarnessResolveKey, projectId ?? null],
    queryFn: async () =>
      (await api.get<ResolveResponse>("/api/pairing/resolve", { params: projectId ? { project_id: projectId } : {} })).data,
  });
  const presence = useFetchPresence();

  if (resolve.isPending || presence.isPending || !resolve.data) {
    return undefined;
  }
  if (!resolve.data.ok) {
    const state = RESOLVE_REASON_TO_STATE[resolve.data.reason ?? ""] ?? "unpaired";
    return { state, message: HARNESS_READINESS_COPY[state] };
  }
  const computerId = resolve.data.computer_id ?? "";
  if (presence.data?.[computerId] !== "connected") {
    return { state: "offline", message: HARNESS_READINESS_COPY.offline };
  }
  return { state: "ready", computerId, provider: resolve.data.provider ?? "", model: resolve.data.model ?? "" };
};

// Comes from the computer's live T3 server, so consumers fall back to manual id entry on error.
export const useFetchHarnessProjects = (computerId: string) =>
  useQuery({
    queryKey: [getHarnessProjectsKey, computerId],
    queryFn: async () => (await api.get<{ projects: HarnessProject[] }>(`/api/pairing/computers/${computerId}/projects`)).data.projects,
    enabled: !!computerId,
    retry: false,
    staleTime: 60_000,
  });

// Same live-server caveat as useFetchHarnessProjects: falls back to manual provider/model entry on error.
export const useFetchHarnessProviders = (computerId: string) =>
  useQuery({
    queryKey: [getHarnessProvidersKey, computerId],
    queryFn: async () => (await api.get<{ providers: HarnessProvider[] }>(`/api/pairing/computers/${computerId}/providers`)).data.providers,
    enabled: !!computerId,
    retry: false,
    staleTime: 60_000,
  });

export const useFetchPairingDefaults = () =>
  useQuery({
    queryKey: [getPairingDefaultsKey],
    queryFn: async () => (await api.get<PairingDefaults>("/api/pairing/defaults")).data,
  });

export const useUpdatePairingDefaults = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: PairingDefaultsFormData) =>
      (await api.put<PairingDefaults>("/api/pairing/defaults", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getPairingDefaultsKey] });
      toast.success("Defaults updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchProjectLink = (projectId: string) =>
  useQuery({
    queryKey: [getProjectLinkKey, projectId],
    queryFn: async () => (await api.get<ProjectLink>(`/api/pairing/projects/${projectId}`)).data,
    enabled: !!projectId,
  });

export const useSetProjectLink = (projectId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProjectLinkFormData) =>
      (await api.put<ProjectLink>(`/api/pairing/projects/${projectId}`, input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectLinkKey, projectId] });
      toast.success("Project link saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useClearProjectLink = (projectId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/pairing/projects/${projectId}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectLinkKey, projectId] });
      toast.success("Project link cleared");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
