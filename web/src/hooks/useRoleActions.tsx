import { useState } from "react";

import { CloneRoleDialog } from "@/components/settings/CloneRoleDialog";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import type { RowAction } from "@/components/settings/RowActionsMenu";
import { useCreateWorkspaceRole, useDeleteWorkspaceRole, useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { copyName, type Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

const deleteMessage = (held: number): string => {
  if (held === 0) return "Nobody holds this role. Deleting it can't be undone.";
  const who = held === 1 ? "1 person loses" : `${held} people lose`;
  return `${who} this role. Anyone left with no role stays a member and sees no project. Deleting it can't be undone.`;
};

// A role's menu (Edit, Duplicate, Clone to workspace, Delete) and the dialogs it opens, for its column head.
export const useRoleActions = (role: Role, workspaceId: string, holders: TeamPerson[] | undefined) => {
  const { data: roles = [] } = useFetchWorkspaceRoles(workspaceId);
  const createRole = useCreateWorkspaceRole(workspaceId);
  const deleteRole = useDeleteWorkspaceRole(workspaceId);
  const { open: confirm } = useConfirmationDialog();
  const [editing, setEditing] = useState(false);
  const [cloning, setCloning] = useState(false);

  const duplicate = () =>
    createRole.mutate({
      name: copyName(role.name, roles.map((r) => r.name)),
      description: role.description,
      actions: role.permissions,
      welcome_steps: role.welcome_steps,
      show_locked_actions: role.show_locked_actions,
    });

  const remove = async () => {
    const ok = await confirm({ title: `Delete ${role.name}?`, message: deleteMessage(holders?.length ?? 0), confirmLabel: "Delete role" });
    if (!ok) return;
    deleteRole.mutate(role.id);
  };

  const actions: RowAction[] = [
    { label: "Edit", permission: "roles:write", verb: "edit roles", onSelect: () => setEditing(true) },
    { label: "Duplicate", permission: "roles:write", verb: "create roles", onSelect: duplicate },
    { label: "Clone to workspace…", permission: "roles:clone", onSelect: () => setCloning(true) },
    { label: "Delete", destructive: true, permission: "roles:write", verb: "delete roles", onSelect: () => void remove() },
  ];

  const dialogs = (
    <>
      <RoleEditorDialog open={editing} role={role} holders={holders} onClose={() => setEditing(false)} />
      {cloning && <CloneRoleDialog role={role} open onClose={() => setCloning(false)} />}
    </>
  );
  return { actions, dialogs };
};
