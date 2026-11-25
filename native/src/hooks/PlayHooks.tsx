import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import type { PlayStage } from "@nexul/client-core/plays";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { runStateChanged } from "@/hooks/TrailHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { defineQuery } from "@/lib/liveQuery";
import { autoPlayRequest, playRequest, type AutoPlay, type ListedPlay, type PlayType, type WorkspacePlay } from "@/models/Play";

export const getWorkspacePlaysKey = "getWorkspacePlays";
export const getApplicablePlaysKey = "getApplicablePlays";
export const getAutoPlaysKey = "getAutoPlays";

// Each row carries what its runs say and how many auto plays are on, so a run's state move and an auto play's change refetch its workspace's list.
const workspacePlaysQuery = defineQuery({
  key: getWorkspacePlaysKey,
  fetch: (workspaceId: string) => api.get<WorkspacePlay[]>(`/api/workspaces/${workspaceId}/plays`),
  refreshes: {
    "play.created": { key: (p) => p.play.workspace_id },
    "play.updated": { key: (p) => p.play.workspace_id },
    "play.deleted": { key: (p) => p.workspace_id },
    "auto_play.created": { key: (p) => p.auto_play.workspace_id },
    "auto_play.updated": { key: (p) => p.auto_play.workspace_id },
    "auto_play.deleted": { key: (p) => p.workspace_id },
    "play.run": {
      patch: (client, frame) => {
        if (runStateChanged(frame)) void client.invalidateQueries({ queryKey: [getWorkspacePlaysKey, frame.workspace_id], exact: true });
      },
    },
  },
});

export const useFetchWorkspacePlays = () => {
  const workspaceId = useCurrentWorkspaceId() ?? "";
  return useQuery({ ...workspacePlaysQuery.options(workspaceId), enabled: workspaceId !== "" });
};

// The plays a ticket or doc offers: enabled plays of that type the viewer may run in the project, at the ticket's stage.
const applicablePlaysQuery = defineQuery({
  key: getApplicablePlaysKey,
  fetch: (workspaceId: string, projectId: string, type: PlayType, stage: PlayStage | "") =>
    api.get<ListedPlay[]>(`/api/workspaces/${workspaceId}/plays/applicable?project_id=${projectId}&type=${type}${stage ? `&stage=${stage}` : ""}`),
  refreshes: {
    "play.created": { key: (p) => p.play.workspace_id },
    "play.updated": { key: (p) => p.play.workspace_id },
    "play.deleted": { key: (p) => p.workspace_id },
  },
});

export const useFetchApplicablePlays = (projectId: string, type: PlayType, stage: PlayStage | undefined) => {
  const workspaceId = useCurrentWorkspaceId() ?? "";
  return useQuery({
    ...applicablePlaysQuery.options(workspaceId, projectId, type, stage ?? ""),
    enabled: workspaceId !== "" && projectId !== "" && (type !== "ticket" || stage !== undefined),
  });
};

const refetchAutoPlays = (client: QueryClient, workspaceId: string, playId: string) =>
  void client.invalidateQueries({ queryKey: [getAutoPlaysKey, workspaceId, playId], exact: true });

// A play's auto plays, oldest first; a frame refetches its own play's list alone.
const autoPlaysQuery = defineQuery({
  key: getAutoPlaysKey,
  fetch: (workspaceId: string, playId: string) => api.get<AutoPlay[]>(`/api/workspaces/${workspaceId}/plays/${playId}/auto-plays`),
  refreshes: {
    "auto_play.created": { patch: (client, p) => refetchAutoPlays(client, p.auto_play.workspace_id, p.auto_play.play_id) },
    "auto_play.updated": { patch: (client, p) => refetchAutoPlays(client, p.auto_play.workspace_id, p.auto_play.play_id) },
    "auto_play.deleted": { patch: (client, p) => refetchAutoPlays(client, p.workspace_id, p.play_id) },
  },
});

export const useFetchAutoPlays = (workspaceId: string, playId: string, enabled: boolean) =>
  useQuery({ ...autoPlaysQuery.options(workspaceId, playId), enabled: enabled && workspaceId !== "" && playId !== "" });

// The switch answers at once; a refused save puts the list back as it was.
export const useSetPlayEnabled = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("plays:write", ({ play, enabled }: { play: WorkspacePlay; enabled: boolean }) =>
      api.patch<unknown>(`/api/workspaces/${play.workspace_id}/plays/${play.id}`, playRequest(play, enabled)),
    ),
    onMutate: ({ play, enabled }) => {
      const queryKey = [getWorkspacePlaysKey, play.workspace_id];
      const previous = client.getQueryData<WorkspacePlay[]>(queryKey);
      client.setQueryData<WorkspacePlay[]>(queryKey, (plays) => plays?.map((p) => (p.id === play.id ? { ...p, enabled } : p)));
      return { previous };
    },
    onError: (_error, { play }, context) => client.setQueryData([getWorkspacePlaysKey, play.workspace_id], context?.previous),
  });
};

export const useSetAutoPlayEnabled = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("autoplays:write", ({ autoPlay, enabled }: { autoPlay: AutoPlay; enabled: boolean }) =>
      api.patch<AutoPlay>(`/api/workspaces/${autoPlay.workspace_id}/plays/${autoPlay.play_id}/auto-plays/${autoPlay.id}`, autoPlayRequest(autoPlay, enabled)),
    ),
    onMutate: ({ autoPlay, enabled }) => {
      const queryKey = [getAutoPlaysKey, autoPlay.workspace_id, autoPlay.play_id];
      const previous = client.getQueryData<AutoPlay[]>(queryKey);
      client.setQueryData<AutoPlay[]>(queryKey, (list) => list?.map((a) => (a.id === autoPlay.id ? { ...a, enabled } : a)));
      return { previous };
    },
    onError: (_error, { autoPlay }, context) => client.setQueryData([getAutoPlaysKey, autoPlay.workspace_id, autoPlay.play_id], context?.previous),
  });
};
