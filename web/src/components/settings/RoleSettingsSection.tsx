import { useState, type FormEvent } from "react";
import { PlusIcon } from "lucide-react";

import { PermissionGrid } from "@/components/access/PermissionGrid";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { RoleRow } from "@/components/settings/RoleRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useCreateWorkspaceRole, useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Gated on roles:write, same gate-in-parent pattern as account management.
export const RoleSettingsSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles, isPending: rolesPending, error: rolesError } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog, isPending: catalogPending, error: catalogError } = useFetchPermissionCatalog();
  const createRole = useCreateWorkspaceRole(workspaceId);

  const isPending = rolesPending || catalogPending;
  const error = rolesError ?? catalogError;

  const [name, setName] = useState("");
  const [actions, setActions] = useState<string[]>([]);

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    await createRole.mutateAsync({ name: trimmed, actions });
    setName("");
    setActions([]);
  };

  return (
    <SettingsCard
      id="roles"
      title="Roles & permissions"
      description="Custom roles this workspace can assign when inviting someone. The Owner role is a
        protected singleton and can't be renamed, edited, or deleted here."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div className="space-y-4">
          {roles.length === 0 && <NoDataDisplay message="No roles yet" />}
          {roles.length > 0 && (
            <ul className="divide-y divide-border overflow-hidden rounded-md border">
              {roles.map((role) => (
                <RoleRow key={role.id} role={role} workspaceId={workspaceId} catalog={catalog} />
              ))}
            </ul>
          )}

          <form onSubmit={onSubmit} className="space-y-3 rounded-md border bg-card p-3 shadow-card">
            <Input
              aria-label="New role name"
              placeholder="New role name"
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
            <PermissionGrid entries={catalog} value={actions} onChange={setActions} />
            <Button type="submit" disabled={createRole.isPending}>
              <PlusIcon className="size-4" />
              Create role
            </Button>
          </form>
        </div>
      )}
    </SettingsCard>
  );
};
