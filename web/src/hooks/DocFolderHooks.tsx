import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import type { Doc } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import { dropRow, type LiveFollower } from "@/lib/live";
import { folderPath } from "@/utils/DocFolderTreeUtility";

export const getDocFoldersKey = "getDocFolders";

// The default folder first, then creation order; a viewer without docs:write gets only folders holding a doc they can open.
export const docFoldersQuery = (projectId: string) =>
  queryOptions({
    queryKey: [getDocFoldersKey, projectId],
    queryFn: async () => (await api.get<DocFolder[]>("/api/docs/folders", { params: { project_id: projectId } })).data,
    enabled: !!projectId,
  });

export const useFetchDocFolders = (projectId: string) => useQuery(docFoldersQuery(projectId));

// A folder's path by id ("Clients / Acme"); a doc whose folder is not listed yet reads as the default folder's, as the list groups it.
export const useDocFolderNames = (projectId: string) => {
  const { data: folders } = useFetchDocFolders(projectId);
  const fallback = folders?.find((f) => f.is_default)?.name ?? "";
  return (folderId: string) => (folders && folderPath(folders, folderId)) || fallback;
};

export const useCreateDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("docs:write", async (input: { projectId: string; parentId?: string | undefined; name: string }) =>
      (await api.post<DocFolder>("/api/docs/folders", { project_id: input.projectId, parent_id: input.parentId ?? "", name: input.name })).data),
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
    mutationFn: requires("docs:write", async (input: { id: string; name: string }) =>
      (await api.put<DocFolder>(`/api/docs/folders/${input.id}`, { name: input.name })).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      toast.success("Folder renamed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// An empty parentId puts the folder at the top level; everything inside it moves with it.
export const useMoveDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("docs:write", async (input: { id: string; parentId: string }) =>
      (await api.post<DocFolder>(`/api/docs/folders/${input.id}/move`, { parent_id: input.parentId })).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      toast.success("Folder moved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The folder goes with every folder inside it; their docs move to its parent, or the default folder; none is deleted.
export const useDeleteDocFolder = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("docs:write", async (id: string) => {
      await api.delete(`/api/docs/folders/${id}`);
    }),
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
    mutationFn: requires("docs:write", async (input: { id: string; folderId: string }) =>
      (await api.post<Doc>(`/api/docs/${input.id}/move`, { folder_id: input.folderId })).data),
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

interface GrantPayload {
  resource_type: string;
}

// Folders list in creation order, so a rename keeps its row's place.
export const docFolderFollower: LiveFollower = {
  // The frame names no project, so a folder's rule refetches every doc and folder list.
  "access.grant.changed": ({ resource_type }: GrantPayload, { client }) =>
    resource_type === "folder" &&
    Promise.all([client.invalidateQueries({ queryKey: [getDocsKey] }), client.invalidateQueries({ queryKey: [getDocFoldersKey] })]),
  "doc.folder.created": ({ folder }: FolderPayload, { client }) =>
    client.invalidateQueries({ queryKey: [getDocFoldersKey, folder.project_id], exact: true }),
  "doc.folder.updated": ({ folder }: FolderPayload, { client }) =>
    client.setQueryData<DocFolder[]>([getDocFoldersKey, folder.project_id], (list) => list?.map((f) => (f.id === folder.id ? folder : f))),
  "doc.folder.deleted": ({ folder }: FolderPayload, { client }) => dropRow(client, [getDocFoldersKey, folder.project_id], folder.id),
};
