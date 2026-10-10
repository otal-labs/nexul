import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";
import { toast } from "sonner";

import { api, errorMessage, type ApiErrorBody } from "@/api/client";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import type { AutoPlay, AutoPlayLimits, SaveAutoPlayRequest } from "@/models/AutoPlay";
import type { Play, PlayStage, PlayType, SavePlayFormData } from "@/models/Play";
import { toSavePlayRequest } from "@/models/Play";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followEach, type LiveFollower } from "@/lib/live";
import { withoutErrorPrefix } from "@/utils/PlayQueueUtility";

export const getWorkspacePlaysKey = "getWorkspacePlays";
export const getApplicablePlaysKey = "getApplicablePlays";
export const getAutoPlaysKey = "getAutoPlays";
export const getAutoPlayLimitsKey = "getAutoPlayLimits";

export const useFetchWorkspacePlays = (workspaceId: string) =>
  useQuery({
    queryKey: [getWorkspacePlaysKey, workspaceId],
    queryFn: async () => (await api.get<Play[]>(`/api/workspaces/${workspaceId}/plays`)).data,
    enabled: workspaceId !== "",
  });

// A play's auto plays, oldest first, for its settings page.
export const useFetchAutoPlays = (workspaceId: string, playId: string) =>
  useQuery({
    queryKey: [getAutoPlaysKey, workspaceId, playId],
    queryFn: async () => (await api.get<AutoPlay[]>(`/api/workspaces/${workspaceId}/plays/${playId}/auto-plays`)).data,
    enabled: workspaceId !== "" && playId !== "",
  });

const autoPlaysPath = (workspaceId: string, playId: string) => `/api/workspaces/${workspaceId}/plays/${playId}/auto-plays`;

export const useCreateAutoPlay = (workspaceId: string, playId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveAutoPlayRequest) => (await api.post<AutoPlay>(autoPlaysPath(workspaceId, playId), input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutoPlaysKey, workspaceId, playId], exact: true });
      toast.success("Auto play added, switched off until you turn it on");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateAutoPlay = (workspaceId: string, playId: string) => {
  const client = useQueryClient();
  const key = [getAutoPlaysKey, workspaceId, playId];
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: SaveAutoPlayRequest }) =>
      (await api.patch<AutoPlay>(`${autoPlaysPath(workspaceId, playId)}/${id}`, input)).data,
    // The switch answers at once; a refused save puts the list back as it was.
    onMutate: async ({ id, input }) => {
      await client.cancelQueries({ queryKey: key, exact: true });
      const previous = client.getQueryData<AutoPlay[]>(key);
      client.setQueryData<AutoPlay[]>(key, (list) => list?.map((a) => (a.id === id ? { ...a, ...input } : a)));
      return { previous };
    },
    onSuccess: (saved, _, context) => {
      const before = context.previous?.find((a) => a.id === saved.id);
      if (before && before.enabled !== saved.enabled) {
        toast.success(saved.enabled ? "Auto play switched on" : "Auto play switched off");
        return;
      }
      toast.success("Auto play saved");
    },
    onError: (error, _, context) => {
      client.setQueryData(key, context?.previous);
      toast.error(errorMessage(error));
    },
    onSettled: () => client.invalidateQueries({ queryKey: key, exact: true }),
  });
};

export const useDeleteAutoPlay = (workspaceId: string, playId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`${autoPlaysPath(workspaceId, playId)}/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutoPlaysKey, workspaceId, playId], exact: true });
      toast.success("Auto play deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const limitsPath = (workspaceId: string) => `/api/workspaces/${workspaceId}/plays/auto-play-limits`;

// The workspace's cap on automatic runs per ticket per day, shared by every play's auto plays.
export const useFetchAutoPlayLimits = (workspaceId: string) =>
  useQuery({
    queryKey: [getAutoPlayLimitsKey, workspaceId],
    queryFn: async () => (await api.get<AutoPlayLimits>(limitsPath(workspaceId))).data,
    enabled: workspaceId !== "",
  });

export const useSetAutoPlayDailyCap = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (cap: number) =>
      (await api.patch<AutoPlayLimits>(limitsPath(workspaceId), { daily_cap_per_ticket: cap })).data,
    onSuccess: (limits) => {
      client.setQueryData([getAutoPlayLimitsKey, workspaceId], limits);
      toast.success(`Each ticket now runs at most ${limits.daily_cap_per_ticket} auto plays a day`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The run buttons a ticket or doc shows: enabled plays of that type the caller may fire on this project, at this stage.
export const useFetchApplicablePlays = (workspaceId: string, projectId: string, type: PlayType, stage: PlayStage | undefined) =>
  useQuery({
    queryKey: [getApplicablePlaysKey, workspaceId, projectId, type, stage ?? null],
    queryFn: async () =>
      (
        await api.get<Play[]>(`/api/workspaces/${workspaceId}/plays/applicable`, {
          params: { project_id: projectId, type, ...(stage ? { stage } : {}) },
        })
      ).data,
    enabled: workspaceId !== "" && projectId !== "" && (type !== "ticket" || stage !== undefined),
  });

// The doc's seeded Clarify via AI and To tickets via AI plays, when the caller may run them on this project.
export const useDocBuiltinPlays = (projectId: string) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchApplicablePlays(workspaceId, projectId, "doc", undefined);
  return {
    clarify: plays?.find((p) => p.builtin_key === "clarify"),
    toTickets: plays?.find((p) => p.builtin_key === "to-tickets-via-ai"),
  };
};

// A ticket's stage is its current column's kind; the rail, the bottom bar and the decisions check notice read this one query.
export const useApplicableTicketPlays = (ticket: Ticket | undefined) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: statuses } = useFetchProjectStatuses(ticket?.project_id);
  const stage = statuses?.find((s) => s.id === ticket?.status)?.kind;
  return useFetchApplicablePlays(workspaceId, ticket?.project_id ?? "", "ticket", stage);
};

// A taken label comes back keyed under the label field; the play dialog shows it there rather than in a toast.
export const playLabelError = (error: unknown): string | undefined => {
  const message = (error as AxiosError<ApiErrorBody> | null)?.response?.data?.errors?.label?.[0];
  return message && withoutErrorPrefix(message);
};

const toastUnlessLabelError = (error: unknown) => {
  if (!playLabelError(error)) toast.error(errorMessage(error));
};

export const useCreatePlay = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: SavePlayFormData) =>
      (await api.post<Play>(`/api/workspaces/${workspaceId}/plays`, toSavePlayRequest(input))).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacePlaysKey, workspaceId] });
      toast.success("Play created");
    },
    onError: toastUnlessLabelError,
  });
};

export const useUpdatePlay = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ playId, input }: { playId: string; input: SavePlayFormData }) =>
      (await api.patch<Play>(`/api/workspaces/${workspaceId}/plays/${playId}`, toSavePlayRequest(input))).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacePlaysKey, workspaceId] });
      toast.success("Play updated");
    },
    onError: toastUnlessLabelError,
  });
};

export const useDeletePlay = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (playId: string) => api.delete(`/api/workspaces/${workspaceId}/plays/${playId}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getWorkspacePlaysKey, workspaceId] });
      toast.success("Play deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A play's lists are its workspace's: the plays page and the run menus.
const refetchWorkspacePlays = (client: QueryClient, workspaceId: string) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getWorkspacePlaysKey, workspaceId], exact: true }),
    client.invalidateQueries({ queryKey: [getApplicablePlaysKey, workspaceId] }),
  ]);

const refetchAutoPlays = (client: QueryClient, workspaceId: string, playId: string) =>
  client.invalidateQueries({ queryKey: [getAutoPlaysKey, workspaceId, playId], exact: true });

export const playFollower: LiveFollower = {
  ...followEach(["play.created", "play.updated"], ({ play }: { play: Play }, { client }) => refetchWorkspacePlays(client, play.workspace_id)),
  "play.deleted": ({ id, workspace_id }: { id: string; workspace_id: string }, { client }) =>
    Promise.all([refetchWorkspacePlays(client, workspace_id), refetchAutoPlays(client, workspace_id, id)]),
  ...followEach(["auto_play.created", "auto_play.updated"], ({ auto_play }: { auto_play: AutoPlay }, { client }) =>
    refetchAutoPlays(client, auto_play.workspace_id, auto_play.play_id),
  ),
  "auto_play.deleted": ({ play_id, workspace_id }: { play_id: string; workspace_id: string }, { client }) =>
    refetchAutoPlays(client, workspace_id, play_id),
  "auto_play.limits_updated": ({ workspace_id, ...limits }: AutoPlayLimits & { workspace_id: string }, { client }) =>
    client.setQueryData([getAutoPlayLimitsKey, workspace_id], limits),
};
