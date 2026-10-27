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
  type PassOver,
  passOverLines,
} from "@/models/Pairing";
import { ownComputers } from "@/models/SharedComputer";
import { notViewing } from "@/stores/viewAsStore";
import { followEach, type LiveFollower } from "@/lib/live";
import type { RunFrame } from "@/models/Trail";

export const getComputersKey = "getComputers";
export const getPairingDefaultsKey = "getPairingDefaults";
export const getHarnessProjectsKey = "getHarnessProjects";
export const getPairingPresenceKey = "getPairingPresence";
export const getHarnessProvidersKey = "getHarnessProviders";
export const getHarnessResolveKey = "getHarnessResolve";
export const getMCPTokenKey = "getMCPToken";

// The NotConfiguredReason values a pressed play's preview can fail with, mapped onto HarnessReadiness states; any other
// reason names one computer that can't take it, which reads like none can.
const RESOLVE_REASON_TO_STATE: Record<string, Exclude<HarnessReadiness["state"], "ready">> = {
  unpaired: "unpaired",
  expired_token: "expired",
  needs_location: "needs_location",
  offline: "offline",
};

interface ResolveResponse {
  ok: boolean;
  reason?: string;
  computer_id?: string;
  harness_project_id?: string;
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
  passed_over?: PassOver[];
}

// The blocked state for a refused preview: its own copy, with every computer's reason after it when the server named them.
const blockedReadiness = (data: ResolveResponse): HarnessReadiness => {
  const state = RESOLVE_REASON_TO_STATE[data.reason ?? ""] ?? "no_computer_ready";
  const copy = HARNESS_READINESS_COPY[state];
  const passes = state === "needs_location" ? [] : (data.passed_over ?? []);
  const message = passes.length > 0 ? `${copy.message} ${passOverLines(passes)}.` : copy.message;
  return { state, message, passedOver: passes, ...(copy.link && { link: copy.link }) };
};

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

// Each change after the first read arrives as computer.presence_changed, which the pairing follower patches in.
export const useFetchPresence = () =>
  useQuery({
    queryKey: [getPairingPresenceKey],
    queryFn: async () => (await api.get<{ computers: Record<string, string> }>("/api/pairing/presence")).data.computers,
    enabled: notViewing,
  });

// Joins the read-only resolve route, where a pressed play would land now (ADR 0165), with presence: ready needs both a
// computer and a connected session.
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
    return blockedReadiness(resolve.data);
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
    passedOver: resolve.data.passed_over ?? [],
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

// The computer to list models from: the first of `ids`, or of the person's order, that is online, because an offline
// one can't answer; the first of them when none is.
export const useModelComputerId = (ids: string[] = []) => {
  const { data: computers } = useListComputers();
  const { data: presence } = useFetchPresence();
  const order = ids.length > 0 ? ids : (computers ?? []).map((c) => c.id);
  const online = (id: string) => presence?.[id] === "connected" || computers?.find((c) => c.id === id)?.runner?.connected === true;
  return order.find(online) ?? order[0] ?? "";
};

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

// The order is the person's alone and every runs-at-once row reads from it, so it lands in the list at once and rolls
// back if the server refuses it.
export const useReorderComputers = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (ids: string[]) => {
      await api.put("/api/pairing/computers/order", { computer_ids: ids });
    }),
    onMutate: async (ids) => {
      await client.cancelQueries({ queryKey: [getComputersKey] });
      const before = client.getQueryData<Computer[]>([getComputersKey]);
      client.setQueryData<Computer[]>([getComputersKey], (old) =>
        old && [...old].sort((a, b) => rank(ids, a.id) - rank(ids, b.id)).map((c) => (c.shared ? c : { ...c, position: ids.indexOf(c.id) })),
      );
      return { before };
    },
    onError: (error, _ids, context) => {
      client.setQueryData([getComputersKey], context?.before);
      toast.error(errorMessage(error));
    },
    onSettled: () => Promise.all([client.invalidateQueries({ queryKey: [getComputersKey] }), client.invalidateQueries({ queryKey: [getHarnessResolveKey] })]),
  });
};

// Shared computers sort after the person's own, as the server lists them.
const rank = (ids: string[], id: string) => {
  const i = ids.indexOf(id);
  return i < 0 ? ids.length : i;
};

export interface ComputerSettingsInput {
  runs_at_once?: number;
  chat_project_id?: string;
}

// A computer's runs at once and chat folder apply the moment they are picked, like the appearance settings.
export const useUpdateComputerSettings = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (input: ComputerSettingsInput) =>
      (await api.patch<Computer>(`/api/pairing/computers/${computerId}`, input)).data),
    onSuccess: (computer) => {
      client.setQueryData<Computer[]>([getComputersKey], (old) => old?.map((c) => (c.id === computer.id ? { ...c, ...pick(computer) } : c)));
      void client.invalidateQueries({ queryKey: [getHarnessResolveKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const pick = (c: Computer) => ({ runs_at_once: c.runs_at_once, chat_project_id: c.chat_project_id });

// Pairing frames name the computer they are about (internal/pairing/events.go).
interface ComputerPayload {
  computer_id: string;
}

// The state is "" once Nexul no longer holds a connection to the computer.
interface PresencePayload extends ComputerPayload {
  state: string;
}

const refetchComputers = (client: QueryClient, readiness: boolean) =>
  Promise.all([client.invalidateQueries({ queryKey: [getComputersKey] }), readiness && client.invalidateQueries({ queryKey: [getHarnessResolveKey] })]);

export const pairingFollower: LiveFollower = {
  // A computer row goes from pairing to paired, loses the tunnel it was paired through, or shows its T3 Code's new version.
  ...followEach(["computer.paired", "computer.harness_switched", "computer.tunnel_removed"], (_p: unknown, { client }) => refetchComputers(client, true)),
  // A computer row and readiness follow its personal runner enrolling, connecting and going away.
  "runner.personal_changed": (_p: unknown, { client }) => refetchComputers(client, true),
  // A computer row follows a failed pairing's reason, and a computer someone shares with the viewer, or stops sharing.
  ...followEach(["computer.pair_failed", "computer.grant_changed"], (_p: unknown, { client }) => refetchComputers(client, false)),
  // A computer row, its readiness and its pickers follow its facts, T3 Code answering among them.
  "computer.facts_changed": ({ computer_id }: ComputerPayload, { client }) =>
    Promise.all([
      refetchComputers(client, true),
      client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
    ]),
  // Readiness follows Nexul's held connection to a computer's T3 Code; the frame carries the whole new state.
  "computer.presence_changed": ({ computer_id, state }: PresencePayload, { client }) =>
    client.setQueryData<Record<string, string>>([getPairingPresenceKey], (old) => {
      if (!old) return old;
      const next = { ...old, [computer_id]: state };
      if (!state) delete next[computer_id];
      return next;
    }),
  // The pickers' "needs setup" tags and the settings dot follow a setup turn confirming or withdrawing a provider.
  ...followEach(["computer.setup_confirmed", "computer.setup_unconfirmed", "computer.setup_finished"], ({ computer_id }: ComputerPayload, { client }) =>
    Promise.all([
      client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computer_id], exact: true }),
      refetchComputers(client, false),
    ]),
  ),
  // A computer's "n of m running" follows a run starting and ending; the frames between carry only its steps.
  "play.run": ({ state, ended_at }: RunFrame, { client }) =>
    (state === "starting" || ended_at !== null) && client.invalidateQueries({ queryKey: [getComputersKey] }),
  // A computer row's MCP token line follows a mint or revoke from setup, the row, or an MCP tool.
  ...followEach(["personal_access_token.minted", "personal_access_token.revoked"], ({ computer_id }: Partial<ComputerPayload>, { client }) =>
    computer_id && client.invalidateQueries({ queryKey: [getMCPTokenKey, computer_id], exact: true }),
  ),
};
