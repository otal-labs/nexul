import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import {
  HARNESS_READINESS_COPY,
  type Computer,
  type HarnessProject,
  type HarnessProvider,
  type HarnessReadiness,
  type PairComputerFormData,
  type PairingDefaults,
  type PairingDefaultsFormData,
  type ProjectLink,
  type ProjectLinkFormData,
} from "@/models/Pairing";

export const getComputersKey = "getComputers";
export const getPairingDefaultsKey = "getPairingDefaults";
export const getProjectLinkKey = "getProjectLink";
export const getHarnessProjectsKey = "getHarnessProjects";
export const getPairingPresenceKey = "getPairingPresence";
export const getHarnessProvidersKey = "getHarnessProviders";
export const getHarnessResolveKey = "getHarnessResolve";

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

export const usePairComputer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: PairComputerFormData) =>
      (await api.post<Computer>("/api/pairing/computers", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success("Computer paired");
    },
    onError: (error) => toast.error(errorMessage(error)),
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
      toast.success("Computer removed");
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
