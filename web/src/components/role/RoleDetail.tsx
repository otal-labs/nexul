import { Pencil } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { Container } from "@/components/Container";
import { RoleDetailHeader } from "@/components/role/RoleDetailHeader";
import { RoleHoldersSection } from "@/components/role/RoleHoldersSection";
import { RoleLevelsSection } from "@/components/role/RoleLevelsSection";
import { RoleSectionLinks } from "@/components/role/RoleSectionLinks";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useSeesTeam } from "@/hooks/AccessHooks";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useRoleActions } from "@/hooks/useRoleActions";
import type { Role } from "@/models/Role";
import { roleHolders } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface RoleDetailProps {
  role: Role;
}

// A role as one page: who holds it and how to give it, then what it allows. Its Edit and menu sit in the header.
export const RoleDetail = ({ role }: RoleDetailProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const seesTeam = useSeesTeam();
  const { data: team } = useFetchTeam(seesTeam);
  const holders = team ? roleHolders(team, workspaceId, role.id) : undefined;
  const { actions, dialogs, edit } = useRoleActions(role, workspaceId, holders);
  const menu = actions.filter((action) => action.label !== "Edit");
  const { data: catalog } = useFetchPermissionCatalog();
  const permissions = role.is_owner_role ? (catalog ?? []).map((entry) => entry.value) : role.permissions;

  return (
    <Container className="space-y-6 pt-0 pb-6 sm:pr-3.5">
      <RoleDetailHeader
        title={role.name}
        line={role.description}
        owner={role.is_owner_role}
        tally={catalog && <RoleSectionLinks catalog={catalog} permissions={permissions} />}
        actions={
          !role.is_owner_role && (
            <>
              <ActionButton permission="roles:write" verb="edit roles" variant="outline" size="sm" icon={Pencil} type="button" onClick={edit}>
                Edit
              </ActionButton>
              <RowActionsMenu subject={role.name} actions={menu} />
            </>
          )
        }
      />
      {seesTeam && <RoleHoldersSection />}
      <RoleLevelsSection role={role} />
      {dialogs}
    </Container>
  );
};
