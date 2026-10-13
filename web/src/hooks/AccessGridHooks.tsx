import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getMyRoleKey, useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AccessGrid, AccessGridSave } from "@/models/AccessGrid";
import { followEach, type LiveFollower } from "@/lib/live";

export const getAccessGridKey = "getAccessGrid";

export const useFetchAccessGrid = (workspaceId: string) =>
  useQuery({
    queryKey: [getAccessGridKey, workspaceId],
    queryFn: async () => (await api.get<AccessGrid>(`/api/workspaces/${workspaceId}/access`)).data,
    enabled: workspaceId !== "",
  });

// No success toast: the save strip answers in its button, and Finish shows its own.
export const useSaveAccessGrid = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: AccessGridSave) => api.put(`/api/workspaces/${workspaceId}/access`, body),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: [getAccessGridKey, workspaceId], exact: true }),
        client.invalidateQueries({ queryKey: [getMyRoleKey, workspaceId], exact: true }),
      ]),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Any rule change, role or membership can move a tick or a holder count; a grant frame names no workspace for a doc,
// play, project or channel rule, so those reread every open grid.
export const accessGridFollower: LiveFollower = {
  "access.grant.changed": ({ resource_type, resource_id }: { resource_type: string; resource_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: resource_type === "workspace" ? [getAccessGridKey, resource_id] : [getAccessGridKey] }),
  ...followEach(["role.updated", "workspace.member.added", "workspace.member.removed", "workspace.member.updated"], ({ workspace_id }: { workspace_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: [getAccessGridKey, workspace_id] }),
  ),
};

// What the one-time access setup asks of the viewer here: the Owner sets it up, a member no project lists yet waits;
// undefined once it is done, or for a member already in a project.
export const useAccessSetupState = (): "owner" | "member" | undefined => {
  const { data } = useFetchMyRole(useWorkspaceStore((s) => s.selectedWorkspaceId));
  if (data?.access_setup_finished !== false) return undefined;
  if (data.roles?.some((role) => role.is_owner)) return "owner";
  if ((data.projects ?? []).length === 0) return "member";
  return undefined;
};

// Whether the viewer holds the Owner role in the selected workspace.
export const useIsWorkspaceOwner = (): boolean => {
  const { data } = useFetchMyRole(useWorkspaceStore((s) => s.selectedWorkspaceId));
  return !!data?.roles?.some((role) => role.is_owner);
};
