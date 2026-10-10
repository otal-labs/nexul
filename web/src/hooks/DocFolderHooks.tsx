import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import type { Doc } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import { dropRow, type LiveFollower } from "@/lib/live";

export const getDocFoldersKey = "getDocFolders";

// The default folder first, then creation order; a viewer without docs:write gets only folders holding a doc they can open.
export const docFoldersQuery = (projectId: string) =>
  queryOptions({
    queryKey: [getDocFoldersKey, projectId],
    queryFn: async () => (await api.get<DocFolder[]>("/api/docs/folders", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useFetchDocFolders = (projectId: string) => useQuery(docFoldersQuery(projectId));

export const useCreateDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { projectId: string; name: string }) =>
      (await api.post<DocFolder>("/api/docs/folders", { project_id: input.projectId, name: input.name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      toast.success("Folder created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRenameDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { id: string; name: string }) =>
      (await api.put<DocFolder>(`/api/docs/folders/${input.id}`, { name: input.name })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      toast.success("Folder renamed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The folder's docs move to the project's default folder; none is deleted.
export const useDeleteDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/docs/folders/${id}`);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      toast.success("Folder deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMoveDoc = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { id: string; folderId: string }) =>
      (await api.post<Doc>(`/api/docs/${input.id}/move`, { folder_id: input.folderId })).data,
    onSuccess: async (_, { id }) => {
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      await client.invalidateQueries({ queryKey: [getDocKey, id] });
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      toast.success("Doc moved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface FolderPayload {
  folder: DocFolder;
}

// Folders list in creation order, so a rename keeps its row's place.
export const docFolderFollower: LiveFollower = {
  "doc.folder.created": ({ folder }: FolderPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getDocFoldersKey, folder.project_id], exact: true }),
  "doc.folder.updated": ({ folder }: FolderPayload, { client }) =>
    client.setQueryData<DocFolder[]>([getDocFoldersKey, folder.project_id], (list) => list?.map((f) => (f.id === folder.id ? folder : f))),
  "doc.folder.deleted": ({ folder }: FolderPayload, { client }) => dropRow(client, [getDocFoldersKey, folder.project_id], folder.id),
};
