import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { RoleMatrix } from "@/components/settings/RoleMatrix";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ActionButton } from "@/components/action/ActionButton";
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

  // Who holds each role; a viewer the server shows no team to just sees no counts.
  const { data: team } = useFetchTeam();
  const holdersOf = (roleId: string) => (team ? roleHolders(team, workspaceId, roleId) : undefined);

  const [creating, setCreating] = useState(false);

  return (
    <SettingsCard
      id="roles"
      title="Roles & permissions"
      description="Every role side by side: what it can open and change in this workspace."
      fill
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
      {roles && roles.length === 0 && <EmptyRow>No roles yet</EmptyRow>}
      {roles && roles.length > 0 && catalog && <RoleMatrix roles={roles} catalog={catalog} workspaceId={workspaceId} holdersOf={holdersOf} />}
      <RoleEditorDialog open={creating} onClose={() => setCreating(false)} />
    </SettingsCard>
  );
};
