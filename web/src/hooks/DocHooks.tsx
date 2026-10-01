import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Doc, DocListItem, DocWatchers, SaveDocFormData } from "@/models/Doc";

export const getDocsKey = "getDocs";
export const getDocKey = "getDoc";
export const getDocWatchersKey = "getDocWatchers";

export const useFetchDocs = () =>
  useQuery({
    queryKey: [getDocsKey],
    queryFn: async () => (await api.get<DocListItem[]>("/api/docs")).data,
  });

// Project-scoped list; the sidebar tree renders each doc directly under its project, no "Docs" row.
export const useFetchDocsByProject = (projectId: string) =>
  useQuery({
    queryKey: [getDocsKey, "byProject", projectId],
    queryFn: async () =>
      (await api.get<DocListItem[]>("/api/docs", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useFetchDoc = (id: string | undefined) =>
  useQuery({
    queryKey: [getDocKey, id],
    queryFn: async () => (await api.get<Doc>(`/api/docs/${id}`)).data,
    enabled: !!id,
  });

export const useCreateDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: SaveDocFormData) => (await api.post<Doc>("/api/docs", input)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      toast.success("Doc created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The doc page edits through its live session; this REST save is the follow-up of a doc created with pasted files.
export const useUpdateDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, title, body }: { id: string; title: string; body: string }) =>
      (await api.put<Doc>(`/api/docs/${id}`, { title, body })).data,
    onSuccess: async (_, vars) => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      await client.invalidateQueries({ queryKey: [getDocKey, vars.id] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useArchiveDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => (await api.post<Doc>(`/api/docs/${id}/archive`)).data,
    onSuccess: async (_, id) => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      await client.invalidateQueries({ queryKey: [getDocKey, id] });
      toast.success("Doc archived");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRestoreDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => (await api.post<Doc>(`/api/docs/${id}/restore`)).data,
    onSuccess: async (_, id) => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      await client.invalidateQueries({ queryKey: [getDocKey, id] });
      toast.success("Doc restored");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetDocLocked = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, locked }: { id: string; locked: boolean }) =>
      (await api.post<Doc>(`/api/docs/${id}/${locked ? "lock" : "unlock"}`)).data,
    onSuccess: async (_, { id, locked }) => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      await client.invalidateQueries({ queryKey: [getDocKey, id] });
      toast.success(locked ? "Doc locked" : "Doc unlocked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// projectId "" duplicates the doc in its own project.
export const useCloneDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, projectId }: { id: string; projectId: string }) =>
      (await api.post<Doc>(`/api/docs/${id}/clone`, { project_id: projectId })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      toast.success("Doc cloned");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/docs/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      toast.success("Doc deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchDocWatchers = (docId: string) =>
  useQuery({
    queryKey: [getDocWatchersKey, docId],
    queryFn: async () => (await api.get<DocWatchers>(`/api/docs/${docId}/watchers`)).data,
    enabled: !!docId,
  });

// Watching is the viewer's own; the response is the doc's watchers as they now stand, so it replaces the cache.
export const useSetDocWatching = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (watching: boolean) => {
      const url = `/api/docs/${docId}/watchers/me`;
      return (watching ? await api.put<DocWatchers>(url) : await api.delete<DocWatchers>(url)).data;
    },
    onSuccess: (watchers) => {
      client.setQueryData([getDocWatchersKey, docId], watchers);
      toast.success(watchers.watching ? "Watching this doc" : "Stopped watching this doc");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
