import { Fragment, useState } from "react";
import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { CreateRoleForm } from "@/components/settings/CreateRoleForm";
import { OwnerRoleRow } from "@/components/settings/OwnerRoleRow";
import { RoleRow } from "@/components/settings/RoleRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Gated on roles:write, same gate-in-parent pattern as account management.
export const RoleSettingsSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles, isPending: rolesPending, error: rolesError } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog, isPending: catalogPending, error: catalogError } = useFetchPermissionCatalog();

  const isPending = rolesPending || catalogPending;
  const error = rolesError ?? catalogError;

  const [creating, setCreating] = useState(false);

  return (
    <SettingsCard
      id="roles"
      title="Roles & permissions"
      description="Custom roles this workspace can assign when inviting someone. The Owner role is a
        protected singleton and can't be renamed, edited, or deleted here."
      footer={
        roles &&
        catalog &&
        !creating && (
          <Button variant="outline" size="sm" onClick={() => setCreating(true)}>
            <PlusIcon className="size-4" />
            New role
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div className="space-y-4">
          {roles.length === 0 && <EmptyRow>No roles yet</EmptyRow>}
          {roles.length > 0 && (
            <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
              {roles.map((role) => (
                <Fragment key={role.id}>
                  {role.is_owner_role && <OwnerRoleRow role={role} />}
                  {!role.is_owner_role && <RoleRow role={role} workspaceId={workspaceId} catalog={catalog} />}
                </Fragment>
              ))}
            </EnterList>
          )}

          {creating && (
            <CreateRoleForm
              workspaceId={workspaceId}
              roles={roles}
              catalog={catalog}
              onDone={() => setCreating(false)}
            />
          )}
        </div>
      )}
    </SettingsCard>
  );
};
