import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { OwnerRoleRow } from "@/components/settings/OwnerRoleRow";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { RoleRow } from "@/components/settings/RoleRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ActionButton } from "@/components/action/ActionButton";
import { useRowGlide } from "@/hooks/useRowGlide";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { roleHolders } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Gated on roles:write, same gate-in-parent pattern as account management.
export const RoleSettingsSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles, isPending: rolesPending, error: rolesError } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog, isPending: catalogPending, error: catalogError } = useFetchPermissionCatalog();

  const isPending = rolesPending || catalogPending;
  const error = rolesError ?? catalogError;

  // Who holds each role; a viewer the server shows no team to just sees no avatars.
  const { data: team } = useFetchTeam();
  const holdersOf = (roleId: string) => (team ? roleHolders(team, workspaceId, roleId) : undefined);

  const [creating, setCreating] = useState(false);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  return (
    <SettingsCard
      id="roles"
      title="Roles & permissions"
      description="What each role can open and change in this workspace. Open a role to see every area."
      footer={
        roles &&
        catalog && (
          <ActionButton permission="roles:write" verb="create roles" icon={PlusIcon} variant="outline" size="sm" onClick={() => setCreating(true)}>
            New role
          </ActionButton>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div className="space-y-4">
          {roles.length === 0 && <EmptyRow>No roles yet</EmptyRow>}
          {roles.length > 0 && (
            <div ref={glideRef}>
              <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
                {roles.filter((role) => role.is_owner_role).map((role) => (
                  <OwnerRoleRow key={role.id} role={role} holders={holdersOf(role.id)} />
                ))}
                {roles.filter((role) => !role.is_owner_role).map((role) => (
                  <RoleRow key={role.id} onLeave={prepareGlide} role={role} workspaceId={workspaceId} catalog={catalog} holders={holdersOf(role.id)} />
                ))}
              </EnterList>
            </div>
          )}
          <RoleEditorDialog open={creating} onClose={() => setCreating(false)} />
        </div>
      )}
    </SettingsCard>
  );
};
