import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleDetailHeader } from "@/components/role/RoleDetailHeader";
import { RoleMatrix } from "@/components/settings/RoleMatrix";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useSeesTeam } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { roleHolders } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Every role side by side, read-only: the role matrix, each head with its holder count and menu.
export const RolesCompareDetail = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles, error: rolesError, isPending: rolesPending } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog, error: catalogError, isPending: catalogPending } = useFetchPermissionCatalog();
  const { data: team } = useFetchTeam(useSeesTeam());
  const error = rolesError ?? catalogError;

  return (
    <Container className="space-y-6 py-6 sm:pr-3.5">
      <RoleDetailHeader title="Compare roles" line="What each role can open and change here, area by area." backAlways />
      {(rolesPending || catalogPending) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div className="rounded-xl border border-border bg-card px-6 pb-6 shadow-card">
          <RoleMatrix roles={roles} catalog={catalog} workspaceId={workspaceId} holdersOf={(roleId) => team && roleHolders(team, workspaceId, roleId)} />
        </div>
      )}
    </Container>
  );
};
