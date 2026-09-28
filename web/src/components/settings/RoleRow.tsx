import { useState, type FocusEvent, type KeyboardEvent } from "react";
import { CopyIcon, CrownIcon, PencilIcon, Trash2 } from "lucide-react";

import { PermissionLevels } from "@/components/access/PermissionLevels";
import { CloneRoleDialog } from "@/components/settings/CloneRoleDialog";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useDeleteWorkspaceRole, useUpdateWorkspaceRole } from "@/hooks/RoleHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { PermissionInfo } from "@/models/Permission";
import { domainsOf, summarize } from "@/models/PermissionLevel";
import type { Role } from "@/models/Role";

interface RoleRowProps {
  role: Role;
  workspaceId: string;
  catalog: PermissionInfo[];
}

// Owner role is protected server-side; this is defense-in-depth, hiding edit/delete too.
export const RoleRow = ({ role, workspaceId, catalog }: RoleRowProps) => {
  const updateRole = useUpdateWorkspaceRole(workspaceId);
  const deleteRole = useDeleteWorkspaceRole(workspaceId);
  const [editing, setEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState(role.name);
  const [actionsDraft, setActionsDraft] = useState<string[]>(role.permissions);
  const [cloning, setCloning] = useState(false);
  const canClone = useHasPermission("roles:clone");

  const summary = summarize(domainsOf(catalog), role.permissions);

  if (role.is_owner_role) {
    return (
      <li className="flex items-center gap-2 bg-card px-3 py-3">
        <span className="flex-1 text-sm font-medium">{role.name}</span>
        <NoFillBadge icon={CrownIcon} color="text-muted-foreground">
          Protected role
        </NoFillBadge>
      </li>
    );
  }

  const startEditing = () => {
    setNameDraft(role.name);
    setActionsDraft(role.permissions);
    setEditing(true);
  };

  // Full-replacement update so an untouched field isn't silently cleared; one atomic commit path.
  const commit = () => {
    const name = nameDraft.trim();
    if (name) {
      void updateRole.mutateAsync({ roleId: role.id, name, actions: actionsDraft });
    }
    setEditing(false);
  };

  const handleGroupBlur = (event: FocusEvent<HTMLDivElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) commit();
  };

  const handleNameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") commit();
    if (event.key === "Escape") setEditing(false);
  };

  return (
    <li className="flex items-center gap-2 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      {editing && (
        <div onBlur={handleGroupBlur} className="flex flex-1 flex-col gap-3">
          <input
            className="min-w-0 rounded-md border border-input px-2 py-1 text-sm"
            aria-label="Role name"
            value={nameDraft}
            autoFocus
            onChange={(event) => setNameDraft(event.target.value)}
            onKeyDown={handleNameKeyDown}
          />
          <PermissionLevels entries={catalog} value={actionsDraft} onChange={setActionsDraft} />
        </div>
      )}
      {!editing && (
        <div className="flex flex-1 flex-col gap-1.5">
          <span className="text-sm font-medium">{role.name}</span>
          <div className="flex flex-wrap gap-x-3 gap-y-1">
            {summary.length === 0 && <span className="text-xs text-muted-foreground">No permissions</span>}
            {summary.map((line) => (
              <NoFillBadge key={line} color="text-muted-foreground">
                {line}
              </NoFillBadge>
            ))}
          </div>
        </div>
      )}
      {!editing && (
        <Button variant="ghost" size="sm" aria-label={`Rename role ${role.name}`} onClick={startEditing}>
          <PencilIcon className="size-4" />
        </Button>
      )}
      {!editing && canClone && (
        <Button
          variant="ghost"
          size="sm"
          aria-label={`Clone ${role.name} to another workspace`}
          title={`Clone ${role.name} to another workspace`}
          onClick={() => setCloning(true)}
        >
          <CopyIcon className="size-4" />
        </Button>
      )}
      {canClone && <CloneRoleDialog role={role} open={cloning} onClose={() => setCloning(false)} />}
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel={`Delete role ${role.name}`}
        disabled={deleteRole.isPending}
        onConfirm={() => deleteRole.mutate(role.id)}
      />
    </li>
  );
};
