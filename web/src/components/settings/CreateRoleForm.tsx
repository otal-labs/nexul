import { useState, type FormEvent } from "react";
import { PlusIcon } from "lucide-react";

import { RoleLevelSections } from "@/components/access/RoleLevelSections";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCreateWorkspaceRole } from "@/hooks/RoleHooks";
import type { PermissionInfo } from "@/models/Permission";
import type { Role } from "@/models/Role";

interface CreateRoleFormProps {
  workspaceId: string;
  roles: Role[];
  catalog: PermissionInfo[];
  onDone: () => void;
}

export const CreateRoleForm = ({ workspaceId, roles, catalog, onDone }: CreateRoleFormProps) => {
  const createRole = useCreateWorkspaceRole(workspaceId);
  const [name, setName] = useState("");
  const [actions, setActions] = useState<string[]>([]);
  const copyable = roles.filter((role) => !role.is_owner_role);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    await createRole.mutateAsync({ name: trimmed, actions });
    onDone();
  };

  const copyFrom = (roleId: string) =>
    setActions(copyable.find((role) => role.id === roleId)?.permissions ?? []);

  return (
    <form
      onSubmit={onSubmit}
      className="animate-in fade-in-0 slide-in-from-top-1 space-y-3 duration-200 ease-out"
    >
      <div className="flex flex-wrap gap-2">
        <Input
          aria-label="New role name"
          placeholder="New role name"
          value={name}
          autoFocus
          onChange={(event) => setName(event.target.value)}
          className="min-w-0 flex-1 basis-48"
        />
        {copyable.length > 0 && (
          <Select onValueChange={copyFrom}>
            <SelectTrigger aria-label="Copy permissions from role" className="basis-40">
              <SelectValue placeholder="Copy from role" />
            </SelectTrigger>
            <SelectContent>
              {copyable.map((role) => (
                <SelectItem key={role.id} value={role.id}>
                  {role.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>
      <RoleLevelSections catalog={catalog} value={actions} onChange={setActions} />
      <div className="flex flex-wrap gap-2">
        <Button type="submit" loading={createRole.isPending}>
          <PlusIcon className="size-4" />
          Create role
        </Button>
        <Button type="button" variant="ghost" onClick={onDone}>
          Cancel
        </Button>
      </div>
    </form>
  );
};
