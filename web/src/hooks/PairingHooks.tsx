import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";
import { toast } from "sonner";

import { api, errorMessage, type ApiErrorBody } from "@/api/client";
import { getPATsKey } from "@/hooks/AuthHooks";
import {
  HARNESS_READINESS_COPY,
  PAIR_FIELDS,
  leftoverSessionNote,
  pairingToken,
  type Computer,
  type CreateComputerTunnelFormData,
  type HarnessProject,
  type HarnessProvider,
  type HarnessReadiness,
  type MCPToken,
  type MintedMCPToken,
  type OptionSetting,
  type PairComputerFormData,
  type PairField,
  type PairingDefaults,
  type PairingDefaultsFormData,
  type TunnelPrerequisite,
  type TunnelStatus,
} from "@/models/Pairing";
import { followEach, type LiveFollower } from "@/lib/live";

export const getComputersKey = "getComputers";
export const getPairingDefaultsKey = "getPairingDefaults";
export const getHarnessProjectsKey = "getHarnessProjects";
export const getPairingPresenceKey = "getPairingPresence";
export const getHarnessProvidersKey = "getHarnessProviders";
export const getHarnessResolveKey = "getHarnessResolve";
export const getTunnelStatusKey = "getTunnelStatus";
export const getTunnelTokenKey = "getTunnelToken";
export const getMCPTokenKey = "getMCPToken";

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
  harness_project_id?: string;
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
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
      const token = pairingToken(form.token);
      if (computerId) return (await api.post<Computer>(`/api/pairing/computers/${computerId}/pair`, { token })).data;
      return (await api.post<Computer>("/api/pairing/computers", { ...form, token })).data;
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

// Read once; every later change arrives as a computer.tunnel_status_changed frame that pairingFollower patches in.
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


// The note carries commands to copy, so it stays until closed.
const sessionNoteToast = (note?: string) => (note ? { description: note, duration: Infinity, closeButton: true } : undefined);

export const useRepairComputer = (computer: Computer) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: PairComputerFormData) =>
      (await api.post<Computer>(`/api/pairing/computers/${computer.id}/repair`, { ...input, token: pairingToken(input.token) })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success("Computer re-paired", sessionNoteToast(leftoverSessionNote(computer, true)));
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (computer: Computer) => {
      await api.delete(`/api/pairing/computers/${computer.id}`);
    },
    onSuccess: async (_, computer) => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      await client.invalidateQueries({ queryKey: [getPATsKey] });
      toast.success("Computer removed", sessionNoteToast(leftoverSessionNote(computer)));
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
      toast.success("MCP token created. Copy it now. It won't be shown again.");
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
  return {
    state: "ready",
    computerId,
    harnessProjectId: resolve.data.harness_project_id ?? "",
    provider: resolve.data.provider ?? "",
    model: resolve.data.model ?? "",
    modelOptions: resolve.data.model_options ?? [],
  };
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
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Pairing frames name the computer they are about (internal/pairing/events.go).
interface ComputerPayload {
  computer_id: string;
}

const refetchComputers = (client: QueryClient, readiness: boolean) =>
  Promise.all([client.invalidateQueries({ queryKey: [getComputersKey] }), readiness && client.invalidateQueries({ queryKey: [getHarnessResolveKey] })]);

export const pairingFollower: LiveFollower = {
  // A computer row goes from pairing to paired, gains or loses its tunnel, or shows its T3 Code's new version.
  ...followEach(["computer.paired", "computer.harness_switched", "computer.tunnel_removed"], (_p: unknown, { client }) => refetchComputers(client, true)),
  "computer.tunnel_created": (_p: unknown, { client }) => refetchComputers(client, false),
  // A computer row follows its personal runner enrolling, connecting and going away.
  "runner.personal_changed": (_p: unknown, { client }) => refetchComputers(client, false),
  // A computer row's details follow its facts, which the frame never carries; its pickers follow what T3 Code lists.
  "computer.facts_changed": ({ computer_id }: ComputerPayload, { client }) =>
    Promise.all([
      refetchComputers(client, false),
      client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
    ]),
  "computer.tunnel_status_changed": (p: ComputerPayload & TunnelStatus, { client }) =>
    client.setQueryData<TunnelStatus>([getTunnelStatusKey, p.computer_id], {
      tunnel: p.tunnel,
      harness_reachable: p.harness_reachable,
      ...(p.harness_version && { harness_version: p.harness_version }),
    }),
  // The pickers' "needs setup" tags follow a setup turn confirming or withdrawing a provider.
  ...followEach(["computer.setup_confirmed", "computer.setup_unconfirmed", "computer.setup_finished"], ({ computer_id }: ComputerPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
  ),
  // A computer row's MCP token line follows a mint or revoke from setup, the row, or an MCP tool.
  ...followEach(["personal_access_token.minted", "personal_access_token.revoked"], ({ computer_id }: Partial<ComputerPayload>, { client }) =>
    computer_id && client.invalidateQueries({ queryKey: [getMCPTokenKey, computer_id], exact: true }),
  ),
};
