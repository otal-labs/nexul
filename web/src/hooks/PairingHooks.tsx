import { queryOptions, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage, type ApiErrorBody } from "@/api/client";
import { getPATsKey } from "@/hooks/AuthHooks";
import {
  HARNESS_READINESS_COPY,
  PAIR_FIELDS,
  leftoverSessionNote,
  type Computer,
  type HarnessProject,
  type HarnessProjects,
  type HarnessProvider,
  type HarnessReadiness,
  type MCPToken,
  type MintedMCPToken,
  type OptionSetting,
  type PairField,
  type PairingDefaults,
  type PairingDefaultsFormData,
} from "@/models/Pairing";
import { ownComputers } from "@/models/SharedComputer";
import { notViewing } from "@/stores/viewAsStore";
import { followEach, type LiveFollower } from "@/lib/live";

export const getComputersKey = "getComputers";
export const getPairingDefaultsKey = "getPairingDefaults";
export const getHarnessProjectsKey = "getHarnessProjects";
export const getPairingPresenceKey = "getPairingPresence";
export const getHarnessProvidersKey = "getHarnessProviders";
export const getHarnessResolveKey = "getHarnessResolve";
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

// One request lists the viewer's computers and those shared with them; each hook selects its half.
export const computersQuery = queryOptions({
  queryKey: [getComputersKey],
  queryFn: async () => (await api.get<{ computers: Computer[] }>("/api/pairing/computers")).data.computers,
});

export const useListComputers = () => useQuery({ ...computersQuery, select: ownComputers, enabled: notViewing });

// The field each failure belongs to, from the error envelope's errors map; empty when the failure has no field.
export const pairFieldErrors = (error: unknown): [PairField, string][] => {
  const errors = (error as AxiosError<ApiErrorBody> | null)?.response?.data?.errors ?? {};
  return PAIR_FIELDS.flatMap((field): [PairField, string][] => {
    const message = errors[field]?.[0];
    return message ? [[field, message]] : [];
  });
};

// The note carries commands to copy, so it stays until closed.
const sessionNoteToast = (note?: string) => (note ? { description: note, duration: Infinity, closeButton: true } : undefined);

export const useDeleteComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (computer: Computer) => {
      await api.delete(`/api/pairing/computers/${computer.id}`);
    }),
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
    mutationFn: requires("none", async () =>
      (await api.post<MintedMCPToken>(`/api/pairing/computers/${computerId}/mcp-token`)).data),
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
    mutationFn: requires("none", async () => {
      await api.delete(`/api/pairing/computers/${computerId}/mcp-token`);
    }),
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
    enabled: notViewing,
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
    return { state, ...HARNESS_READINESS_COPY[state] };
  }
  const computerId = resolve.data.computer_id ?? "";
  if (presence.data?.[computerId] === "rejected") {
    return { state: "expired", ...HARNESS_READINESS_COPY.expired };
  }
  if (presence.data?.[computerId] !== "connected") {
    return { state: "offline", ...HARNESS_READINESS_COPY.offline };
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

// One request reads the computer's T3 projects and whether its session may clone; each hook selects its half.
const harnessProjectsQuery = (computerId: string) =>
  queryOptions({
    queryKey: [getHarnessProjectsKey, computerId],
    queryFn: async () => (await api.get<HarnessProjects>(`/api/pairing/computers/${computerId}/projects`)).data,
    enabled: !!computerId,
    retry: false,
    staleTime: 60_000,
  });

// Comes from the computer's live T3 server, so consumers fall back to manual id entry on error.
export const useFetchHarnessProjects = (computerId: string) =>
  useQuery({ ...harnessProjectsQuery(computerId), select: (data) => data.projects });

// False for a computer paired before Nexul could clone there; it needs pairing again.
export const useCanCloneOnComputer = (computerId: string) =>
  useQuery({ ...harnessProjectsQuery(computerId), select: (data) => data.can_clone });

// The new project lands in the cached list at once, since T3 registers it before the clone finishes.
const useAddHarnessProject = <V,>(computerId: string, body: (input: V) => object, message: (project: HarnessProject, input: V) => string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (input: V) => (await api.post<HarnessProject>(`/api/pairing/computers/${computerId}/projects`, body(input))).data),
    onSuccess: (project, input) => {
      client.setQueryData<HarnessProjects>([getHarnessProjectsKey, computerId], (old) => old && { ...old, projects: [...old.projects, project] });
      void client.invalidateQueries({ queryKey: [getHarnessProjectsKey, computerId], exact: true });
      toast.success(message(project, input));
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCloneHarnessProject = (computerId: string) =>
  useAddHarnessProject(computerId, (repository: string) => ({ repository }), (project, repository) => `Cloning ${repository} into ${project.path}`);

export const useCreateEmptyHarnessProject = (computerId: string) =>
  useAddHarnessProject(computerId, (name: string) => ({ name }), (project) => `Created ${project.title} at ${project.path}`);

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
    mutationFn: requires("none", async (input: PairingDefaultsFormData) =>
      (await api.put<PairingDefaults>("/api/pairing/defaults", input)).data),
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
  // A computer row goes from pairing to paired, loses the tunnel it was paired through, or shows its T3 Code's new version.
  ...followEach(["computer.paired", "computer.harness_switched", "computer.tunnel_removed"], (_p: unknown, { client }) => refetchComputers(client, true)),
  // A computer row follows its personal runner enrolling, connecting and going away, and a failed pairing's reason.
  // ...and a computer someone shares with the viewer, or stops sharing, joining or leaving their list.
  ...followEach(["runner.personal_changed", "computer.pair_failed", "computer.grant_changed"], (_p: unknown, { client }) => refetchComputers(client, false)),
  // A computer row's details follow its facts, which the frame never carries; its pickers follow what T3 Code lists.
  "computer.facts_changed": ({ computer_id }: ComputerPayload, { client }) =>
    Promise.all([
      refetchComputers(client, false),
      client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
    ]),
  // The pickers' "needs setup" tags follow a setup turn confirming or withdrawing a provider.
  ...followEach(["computer.setup_confirmed", "computer.setup_unconfirmed", "computer.setup_finished"], ({ computer_id }: ComputerPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
  ),
  // A computer row's MCP token line follows a mint or revoke from setup, the row, or an MCP tool.
  ...followEach(["personal_access_token.minted", "personal_access_token.revoked"], ({ computer_id }: Partial<ComputerPayload>, { client }) =>
    computer_id && client.invalidateQueries({ queryKey: [getMCPTokenKey, computer_id], exact: true }),
  ),
};
