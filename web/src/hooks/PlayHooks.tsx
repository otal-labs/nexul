import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import type { Play, PlayStage, PlayType, SavePlayFormData } from "@/models/Play";
import { toSavePlayRequest } from "@/models/Play";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getWorkspacePlaysKey = "getWorkspacePlays";
export const getApplicablePlaysKey = "getApplicablePlays";

export const useFetchWorkspacePlays = (workspaceId: string) =>
  useQuery({
    queryKey: [getWorkspacePlaysKey, workspaceId],
    queryFn: async () => (await api.get<Play[]>(`/api/workspaces/${workspaceId}/plays`)).data,
    enabled: workspaceId !== "",
  });

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

// A ticket's stage is its current column's kind; the rail and the bottom bar both read this one query.
export const useApplicableTicketPlays = (ticket: Ticket) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const stage = statuses?.find((s) => s.id === ticket.status)?.kind;
  return useFetchApplicablePlays(workspaceId, ticket.project_id, "ticket", stage);
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
    onError: (error) => toast.error(errorMessage(error)),
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
    onError: (error) => toast.error(errorMessage(error)),
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
