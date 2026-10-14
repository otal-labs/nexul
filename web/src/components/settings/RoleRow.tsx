import { useState } from "react";
import { ChevronRightIcon } from "lucide-react";

import { CloneRoleDialog } from "@/components/settings/CloneRoleDialog";
import { RoleAccessDetail } from "@/components/settings/RoleAccessDetail";
import { RoleAccessSummary } from "@/components/settings/RoleAccessSummary";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { RoleHolders } from "@/components/settings/RoleHolders";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useCreateWorkspaceRole, useDeleteWorkspaceRole, useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { cn } from "@/lib/utils";
import type { PermissionInfo } from "@/models/Permission";
import { copyName, type Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

interface RoleRowProps {
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
  role: Role;
  workspaceId: string;
  catalog: PermissionInfo[];
  holders: TeamPerson[] | undefined;
}

const deleteMessage = (held: number): string => {
  if (held === 0) return "Nobody holds this role. Deleting it can't be undone.";
  const who = held === 1 ? "1 person loses" : `${held} people lose`;
  return `${who} this role. Anyone left with no role stays a member and sees no project. Deleting it can't be undone.`;
};

export const RoleRow = ({ onLeave, role, workspaceId, catalog, holders }: RoleRowProps) => {
  const { data: roles = [] } = useFetchWorkspaceRoles(workspaceId);
  const createRole = useCreateWorkspaceRole(workspaceId);
  const deleteRole = useDeleteWorkspaceRole(workspaceId);
  const { open: confirm } = useConfirmationDialog();
  const [editing, setEditing] = useState(false);
  const [open, setOpen] = useState(false);
  const [cloning, setCloning] = useState(false);
  const [leaving, setLeaving] = useState(false);

  const duplicate = () =>
    createRole.mutate({
      name: copyName(role.name, roles.map((r) => r.name)),
      description: role.description,
      actions: role.permissions,
      welcome_steps: role.welcome_steps,
      show_locked_actions: role.show_locked_actions,
    });

  const remove = async () => {
    const held = holders?.length ?? 0;
    const ok = await confirm({
      title: `Delete ${role.name}?`,
      message: deleteMessage(held),
      confirmLabel: "Delete role",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave?.();
    deleteRole.mutate(role.id, { onError: () => setLeaving(false) });
  };

  const actions: RowAction[] = [
    { label: "Edit", permission: "roles:write", verb: "edit roles", onSelect: () => setEditing(true) },
    { label: "Duplicate", permission: "roles:write", verb: "create roles", onSelect: duplicate },
    { label: "Clone to workspace…", permission: "roles:clone", onSelect: () => setCloning(true) },
    { label: "Delete", destructive: true, permission: "roles:write", onSelect: () => void remove() },
  ];

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "space-y-2.5 bg-card px-4 py-3.5")}>
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
      <RoleEditorDialog open={editing} role={role} holders={holders} onClose={() => setEditing(false)} />
      {cloning && <CloneRoleDialog role={role} open onClose={() => setCloning(false)} />}
    </li>
  );
};
