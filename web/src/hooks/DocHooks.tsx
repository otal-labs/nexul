import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Doc, DocListItem, SaveDocFormData } from "@/models/Doc";

export const getDocsKey = "getDocs";
export const getDocKey = "getDoc";

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

