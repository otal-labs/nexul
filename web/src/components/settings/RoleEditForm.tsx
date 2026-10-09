import { useState, type FormEvent, type KeyboardEvent } from "react";

import { RoleLevelSections } from "@/components/access/RoleLevelSections";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useUpdateWorkspaceRole } from "@/hooks/RoleHooks";
import type { PermissionInfo } from "@/models/Permission";
import type { Role } from "@/models/Role";

interface RoleEditFormProps {
  role: Role;
  workspaceId: string;
  catalog: PermissionInfo[];
  onDone: () => void;
}

// Full-replacement update so an untouched field isn't silently cleared; one atomic commit path.
export const RoleEditForm = ({ role, workspaceId, catalog, onDone }: RoleEditFormProps) => {
  const updateRole = useUpdateWorkspaceRole(workspaceId);
  const [name, setName] = useState(role.name);
  const [actions, setActions] = useState<string[]>(role.permissions);

  const commit = async (event: FormEvent) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    await updateRole.mutateAsync({ roleId: role.id, name: trimmed, actions });
    onDone();
  };

  const onNameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") onDone();
  };

  return (
    <form onSubmit={(event) => void commit(event)} className="flex min-w-0 flex-1 flex-col gap-4">
      <Input
        aria-label="Role name"
        value={name}
        autoFocus
        onChange={(event) => setName(event.target.value)}
        onKeyDown={onNameKeyDown}
        className="h-9 max-w-sm"
      />
      <RoleLevelSections catalog={catalog} value={actions} onChange={setActions} />
      <div className="flex gap-2">
        <Button type="submit" size="sm" loading={updateRole.isPending}>
          Save role
        </Button>
        <Button type="button" variant="ghost" size="sm" onClick={onDone}>
          Cancel
        </Button>
      </div>
    </form>
  );
};
