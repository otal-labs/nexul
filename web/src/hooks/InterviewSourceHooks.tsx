import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchApplicablePlays, useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import {
  DRAFT_PLAY_KEY,
  type AddInterviewSourceInput,
  type InterviewDraft,
  type InterviewSource,
  type SourceStance,
} from "@/models/InterviewSource";

export const getInterviewSourcesKey = "getInterviewSources";
export const getInterviewDraftsKey = "getInterviewDrafts";

export const useFetchInterviewSources = (projectId: string) =>
  useQuery({
    queryKey: [getInterviewSourcesKey, projectId],
    queryFn: async () =>
      (await api.get<InterviewSource[]>("/api/memories/interview-sources", { params: { project_id: projectId } })).data,
    enabled: projectId !== "",
  });

export const useFetchInterviewDrafts = (projectId: string) =>
  useQuery({
    queryKey: [getInterviewDraftsKey, projectId],
    queryFn: async () =>
      (await api.get<InterviewDraft[]>("/api/memories/interview-drafts", { params: { project_id: projectId } })).data,
    enabled: projectId !== "",
  });

export const useAddInterviewSource = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: AddInterviewSourceInput) =>
      (await api.post<InterviewSource>("/api/memories/interview-sources", input)).data,
    onSuccess: async (source) => {
      await client.invalidateQueries({ queryKey: [getInterviewSourcesKey, source.project_id] });
      toast.success("Source added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The segmented control moves at once; a refusal puts it back.
export const useSetInterviewSourceStance = (projectId: string) => {
  const client = useQueryClient();
  const key = [getInterviewSourcesKey, projectId];
  const patch = (id: string, stance: SourceStance) =>
    client.setQueryData<InterviewSource[]>(key, (list) => list?.map((s) => (s.id === id ? { ...s, stance } : s)));
  return useMutation({
    mutationFn: async ({ id, stance }: { id: string; stance: SourceStance; was: SourceStance }) =>
      (await api.patch<InterviewSource>(`/api/memories/interview-sources/${id}`, { stance })).data,
    onMutate: async ({ id, stance }) => {
      await client.cancelQueries({ queryKey: key });
      patch(id, stance);
    },
    onError: (error, { id, was }) => {
      patch(id, was);
      toast.error(errorMessage(error));
    },
    onSettled: async () => {
      await client.invalidateQueries({ queryKey: key });
    },
  });
};

export const useRemoveInterviewSource = (projectId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/memories/interview-sources/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInterviewSourcesKey, projectId] });
      toast.success("Source removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDismissInterviewDraft = (projectId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/memories/interview-drafts/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getInterviewDraftsKey, projectId] });
      toast.success("Suggestion dismissed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The project's two interview plays: the drafting run by its builtin key, the follow-up run as any other.
export const useInterviewPlays = (projectId: string) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchApplicablePlays(workspaceId, projectId, "interview", undefined);
  return {
    followUp: plays?.find((p) => p.builtin_key !== DRAFT_PLAY_KEY),
    draft: plays?.find((p) => p.builtin_key === DRAFT_PLAY_KEY),
  };
};

// The interview target's trails split by play, newest first like the list they come from; the workspace's play list names the
// drafting play for someone who cannot run it.
export const useInterviewTrails = (projectId: string) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: trails } = useFetchTrails("interview", projectId);
  const { data: plays } = useFetchWorkspacePlays(workspaceId);
  const runnable = useInterviewPlays(projectId).draft;
  const draftId = runnable?.id ?? plays?.find((p) => p.builtin_key === DRAFT_PLAY_KEY)?.id;
  return {
    followUp: trails?.filter((t) => t.play_id !== draftId),
    drafting: trails?.filter((t) => t.play_id === draftId),
  };
};
