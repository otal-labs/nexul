import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Import } from "@/models/Import";
import type { LiveFollower } from "@/lib/live";

export const getImportsKey = "getImports";

// The workspace's latest imports, newest first; the Owner alone may read them.
export const useFetchImports = (workspaceId: string | undefined) =>
  useQuery({
    queryKey: [getImportsKey, workspaceId],
    queryFn: async () =>
      (
        await api.get<Import[]>("/api/imports", {
          params: { workspace_id: workspaceId },
        })
      ).data,
    enabled: !!workspaceId,
  });

// One import of the selected workspace, read from its cached list, which the live follower keeps current.
export const useImport = (id: string): Import | undefined => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useFetchImports(workspaceId).data?.find((imp) => imp.id === id);
};

// Puts an import into its workspace's list in place, or first when the list lacks it.
const putImport = (client: QueryClient, imp: Import) =>
  client.setQueryData<Import[]>([getImportsKey, imp.workspace_id], (list) => {
    if (!list) return list;
    if (!list.some((i) => i.id === imp.id)) return [imp, ...list];
    return list.map((i) => (i.id === imp.id ? imp : i));
  });

export interface StartImportInput {
  source: string;
  project_id: string;
  from: string;
}

export const useStartImport = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (input: StartImportInput) => (await api.post<Import>("/api/imports", input)).data),
    onSuccess: (imp) => putImport(client, imp),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useConfirmImport = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (id: string) => (await api.post<Import>(`/api/imports/${id}/confirm`)).data),
    onSuccess: (imp) => putImport(client, imp),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCancelImport = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (id: string) => (await api.post<Import>(`/api/imports/${id}/cancel`)).data),
    onSuccess: (imp) => {
      putImport(client, imp);
      toast.success("Import cancelled");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The frame carries the whole import, so it patches the list and refetches nothing.
export const importFollower: LiveFollower = {
  "import.progress": ({ import: imp }: { import: Import }, { client }) => putImport(client, imp),
};
