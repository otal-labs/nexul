import { useState } from "react";
import { ChevronRightIcon } from "lucide-react";

import { CloneRoleDialog } from "@/components/settings/CloneRoleDialog";
import { RoleAccessDetail } from "@/components/settings/RoleAccessDetail";
import { RoleAccessSummary } from "@/components/settings/RoleAccessSummary";
import { RoleEditForm } from "@/components/settings/RoleEditForm";
import { RoleHolders } from "@/components/settings/RoleHolders";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useCreateWorkspaceRole, useDeleteWorkspaceRole, useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { copyName, type Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

interface RoleRowProps {
  role: Role;
  workspaceId: string;
  catalog: PermissionInfo[];
  holders: TeamPerson[] | undefined;
}

export const RoleRow = ({ role, workspaceId, catalog, holders }: RoleRowProps) => {
  const { data: roles = [] } = useFetchWorkspaceRoles(workspaceId);
  const createRole = useCreateWorkspaceRole(workspaceId);
  const deleteRole = useDeleteWorkspaceRole(workspaceId);
  const { open: confirm } = useConfirmationDialog();
  const canClone = useHasPermission("roles:clone");
  const [editing, setEditing] = useState(false);
  const [open, setOpen] = useState(false);
  const [cloning, setCloning] = useState(false);

  const duplicate = () =>
    createRole.mutate({ name: copyName(role.name, roles.map((r) => r.name)), actions: role.permissions });

  // A role someone holds can't go: the confirm says who to move first instead of offering a delete that fails.
  const remove = async () => {
    const held = holders?.length ?? 0;
    if (held > 0) {
      await confirm({
        title: `${role.name} is in use`,
        message: `${held === 1 ? "1 person holds" : `${held} people hold`} this role. Give them another role in Team, then delete it.`,
        confirmLabel: "Got it",
        destructive: false,
      });
      return;
    }
    const ok = await confirm({
      title: `Delete ${role.name}?`,
      message: "Nobody holds this role. Deleting it can't be undone.",
      confirmLabel: "Delete role",
    });
    if (ok) deleteRole.mutate(role.id);
  };

  const actions: RowAction[] = [
    { label: "Edit", onSelect: () => setEditing(true) },
    { label: "Duplicate", onSelect: duplicate },
    ...(canClone ? [{ label: "Clone to workspace…", onSelect: () => setCloning(true) }] : []),
    { label: "Delete", destructive: true, onSelect: () => void remove() },
  ];

  return (
    <li className="bg-card px-4 py-3.5">
      {editing && (
        <div className="settle-in">
          <RoleEditForm role={role} workspaceId={workspaceId} catalog={catalog} onDone={() => setEditing(false)} />
        </div>
      )}
      {!editing && (
        <div className="space-y-2.5">
          <div className="flex items-center gap-3">
            <button
              type="button"
              aria-expanded={open}
              onClick={() => setOpen(!open)}
              className="group -ml-1 flex min-w-0 flex-1 items-center gap-1.5 rounded-md px-1 py-0.5 text-left focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <ChevronRightIcon
                aria-hidden
                className={cn(
                  "size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground",
                  open && "rotate-90",
                )}
              />
              <span className="truncate text-sm font-medium" title={role.name}>
                {role.name}
              </span>
            </button>
            {holders && <RoleHolders people={holders} />}
            <RowActionsMenu subject={role.name} actions={actions} />
          </div>
          <div className="pl-6">
            <RoleAccessSummary catalog={catalog} permissions={role.permissions} />
          </div>
          <div inert={!open} aria-hidden={!open} data-closed={!open || undefined} className="disclosure">
            <div className="pl-6">
              <RoleAccessDetail catalog={catalog} permissions={role.permissions} />
            </div>
          </div>
        </div>
      )}
      {canClone && <CloneRoleDialog role={role} open={cloning} onClose={() => setCloning(false)} />}
    </li>
  );
};
