import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchApplicablePlays, useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { AUDIT_KEY, DRAFT_INTERVIEW_KEY } from "@/models/Play";
import type { AddInterviewSourceInput, InterviewDraft, InterviewSource, SourceStance } from "@/models/InterviewSource";
import { followEach, refetchHolding, type LiveFollower } from "@/lib/live";

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
    followUp: plays?.find((p) => p.builtin_key !== DRAFT_INTERVIEW_KEY && p.builtin_key !== AUDIT_KEY),
    draft: plays?.find((p) => p.builtin_key === DRAFT_INTERVIEW_KEY),
    audit: plays?.find((p) => p.builtin_key === AUDIT_KEY),
  };
};

// The interview target's trails split by play, newest first like the list they come from; the workspace's play list names the
// drafting and audit plays for someone who cannot run them.
export const useInterviewTrails = (projectId: string) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: trails } = useFetchTrails("interview", projectId);
  const { data: plays } = useFetchWorkspacePlays(workspaceId);
  const runnable = useInterviewPlays(projectId);
  const draftId = runnable.draft?.id ?? plays?.find((p) => p.builtin_key === DRAFT_INTERVIEW_KEY)?.id;
  const auditId = runnable.audit?.id ?? plays?.find((p) => p.builtin_key === AUDIT_KEY)?.id;
  return {
    followUp: trails?.filter((t) => t.play_id !== draftId && t.play_id !== auditId),
    drafting: trails?.filter((t) => t.play_id === draftId),
    audit: trails?.filter((t) => t.play_id === auditId),
  };
};

// Source and draft frames name the project whose interview they belong to (internal/memories/events.go).
interface InterviewPayload {
  project_id: string;
}

// A source names its doc or memory and dates its change, so only a project list that holds it refetches.
const refetchNaming = (client: QueryClient, projectId: string, ref: string) =>
  refetchHolding(client, [getInterviewSourcesKey, projectId], (sources: InterviewSource[]) => sources.some((s) => s.ref === ref));

export const interviewSourceFollower: LiveFollower = {
  ...followEach(["interview_source.added", "interview_source.changed", "interview_source.removed"], ({ project_id }: InterviewPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getInterviewSourcesKey, project_id], exact: true }),
  ),
  ...followEach(["interview_draft.saved", "interview_draft.dismissed"], ({ project_id }: InterviewPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getInterviewDraftsKey, project_id], exact: true }),
  ),
  "doc.updated": ({ doc }: { doc: { id: string; project_id: string } }, { client }) => refetchNaming(client, doc.project_id, doc.id),
  "memory.updated": ({ memory }: { memory: { id: string; project_id: string } }, { client }) => refetchNaming(client, memory.project_id, memory.id),
  "memory.deleted": ({ id, project_id }: { id: string; project_id: string }, { client }) => refetchNaming(client, project_id, id),
};
