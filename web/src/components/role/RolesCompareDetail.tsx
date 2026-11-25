import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleDetailHeader } from "@/components/role/RoleDetailHeader";
import { RolesShownMenu } from "@/components/role/RolesShownMenu";
import { RoleMatrix } from "@/components/settings/RoleMatrix";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useSeesTeam } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { shownRoles, useComparedRolesStore } from "@/stores/comparedRolesStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { roleHolders } from "@/models/Team";

// The roles side by side, read-only: the role matrix, each head with its holder count and menu, narrowed to the roles ticked in its head.
export const RolesCompareDetail = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles, error: rolesError, isPending: rolesPending } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog, error: catalogError, isPending: catalogPending } = useFetchPermissionCatalog();
  const { data: team } = useFetchTeam(useSeesTeam());
  const hidden = useComparedRolesStore((s) => s.hidden);
  const compared = roles?.filter((role) => !role.is_owner_role) ?? [];
  const error = rolesError ?? catalogError;

  return (
    <Container className="space-y-5 pt-0 pb-6 sm:pr-3.5">
      <RoleDetailHeader title="Compare roles" line="The Owner holds everything, so it isn't listed." backAlways />
      {(rolesPending || catalogPending) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div className="overflow-hidden rounded-lg border border-border bg-card px-6 pb-6 shadow-card">
          <RoleMatrix
            stableGutter={false}
            hideOwner
            roles={shownRoles(compared, hidden)}
            areaHead={<RolesShownMenu roles={compared} />}
            catalog={catalog}
            workspaceId={workspaceId}
            holdersOf={(roleId) => team && roleHolders(team, workspaceId, roleId)}
          />
        </div>
      )}
    </Container>
  );
};
