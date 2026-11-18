import { ShieldIcon } from "lucide-react";
import { useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { NoRolesDetail } from "@/components/role/NoRolesDetail";
import { RoleDetail } from "@/components/role/RoleDetail";
import { RolesCompareDetail } from "@/components/role/RolesCompareDetail";
import { RolesListPane } from "@/components/role/RolesListPane";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useSeesTeam } from "@/hooks/AccessHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { COMPARE_ROLES, NO_ROLES } from "@/models/Role";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The Access page's Roles and people tab: the roles beside the open one, which shows who holds it and what it allows (ADR 0198).
export const RolesAndPeopleSection = () => {
  const { roleId } = useParams();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const workspace = useSelectedWorkspace();
  const seesTeam = useSeesTeam();
  const { data: roles, error, isPending } = useFetchWorkspaceRoles(workspaceId);
  const selected = roles?.find((role) => role.id === roleId);
  const count = roles?.length ?? 0;
  const special = roleId === NO_ROLES && seesTeam;

  return (
    <div className="min-h-0 flex-1">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && roleId === COMPARE_ROLES && (
        <div data-pane-layout className="h-full">
          <div className="panel h-full min-h-0 overflow-hidden">
            <div className="scroll-edge relative my-(--radius-xl) h-[calc(100%-2*var(--radius-xl))] overflow-y-auto [scrollbar-gutter:stable]">
              <RolesCompareDetail />
            </div>
          </div>
        </div>
      )}
      {roles && roleId !== COMPARE_ROLES && (
        <ListDetailLayout
          hasSelection={!!roleId}
          stableGutter
          list={<RolesListPane roles={roles} selectedId={roleId} />}
          placeholder={
            <ListDetailPlaceholder icon={ShieldIcon} title="Select a role" summary={`${count} ${count === 1 ? "role" : "roles"} in ${workspace?.name ?? "this workspace"}`} />
          }
          detail={
            <>
              {selected && <RoleDetail key={selected.id} role={selected} />}
              {roleId === NO_ROLES && seesTeam && <NoRolesDetail />}
              {!selected && !special && <NoDataDisplay message="That role isn't here" />}
            </>
          }
        />
      )}
    </div>
  );
};
