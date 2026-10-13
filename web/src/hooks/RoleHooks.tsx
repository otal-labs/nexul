import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { domainsOf } from "@/models/PermissionLevel";
import type { Role } from "@/models/Role";
import { roleChoices, type RoleChoice } from "@/models/RoleChoice";
import { getTeamKey } from "@/models/Team";
import type { Workspace } from "@/models/Workspace";
import type { LiveFollower } from "@/lib/live";

export const getWorkspaceRolesKey = "getWorkspaceRoles";

// The invite dropdown's source, so invites assign from the workspace's own catalog, never a hardcoded pair.
export const useFetchWorkspaceRoles = (workspaceId: string) =>
  useQuery({
    queryKey: [getWorkspaceRolesKey, workspaceId],
    queryFn: async () => (await api.get<Role[]>(`/api/workspaces/${workspaceId}/roles`)).data,
    enabled: workspaceId !== "",
  });

// The workspace's roles as a checklist reads them: which the viewer may give, judged against every role they hold.
export const useRoleChoices = (workspaceId: string): { choices: RoleChoice[] | undefined; error: Error | null } => {
  const roles = useFetchWorkspaceRoles(workspaceId);
  const me = useFetchMyRole(workspaceId);
  const catalog = useFetchPermissionCatalog();
  const choices = useMemo(
    () => roles.data && me.data && catalog.data && roleChoices(roles.data, me.data.roles ?? [], domainsOf(catalog.data)),
    [roles.data, me.data, catalog.data],
  );
  return { choices, error: roles.error ?? me.error ?? catalog.error };
};

// `actions` must match the /api/permissions/catalog vocabulary exactly — unknowns drop silently server-side.
export const useCreateWorkspaceRole = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ name, actions }: { name: string; actions: string[] }) =>
      (await api.post<Role>(`/api/workspaces/${workspaceId}/roles`, { name, actions })).data,
    onSuccess: async (role) => {
      await client.invalidateQueries({ queryKey: [getWorkspaceRolesKey, workspaceId] });
      toast.success(`${role.name} created`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateWorkspaceRole = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ roleId, name, actions }: { roleId: string; name: string; actions: string[] }) =>
      (await api.patch<Role>(`/api/workspaces/${workspaceId}/roles/${roleId}`, { name, actions })).data,
    onSuccess: async (role) => {
      await client.invalidateQueries({ queryKey: [getWorkspaceRolesKey, workspaceId] });
      toast.success(`${role.name} saved`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteWorkspaceRole = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (roleId: string) =>
      api.delete(`/api/workspaces/${workspaceId}/roles/${roleId}`),
    onSuccess: async () => {
      // Its holders lose it, and the delete's frame is role.updated, which the Team doesn't follow.
      await Promise.all([
        client.invalidateQueries({ queryKey: [getWorkspaceRolesKey, workspaceId] }),
        client.invalidateQueries({ queryKey: [getTeamKey], exact: true }),
      ]);
      toast.success("Role deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface CloneWorkspaceRoleInput {
  role: Role;
  target: Workspace;
}

// The toast names the copy's new name only when the target already had a role by that name.
export const useCloneWorkspaceRole = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ role, target }: CloneWorkspaceRoleInput) =>
      (await api.post<Role>(`/api/workspaces/${role.workspace_id}/roles/${role.id}/clone`, { workspace_id: target.id }))
        .data,
    onSuccess: async (clone, { role, target }) => {
      await client.invalidateQueries({ queryKey: [getWorkspaceRolesKey, target.id] });
      const renamed = clone.name !== role.name ? ` as ${clone.name}` : "";
      toast.success(`Cloned ${role.name} to ${target.name}${renamed}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const roleFollower: LiveFollower = {
  "role.updated": ({ workspace_id }: { workspace_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: [getWorkspaceRolesKey, workspace_id], exact: true }),
};
