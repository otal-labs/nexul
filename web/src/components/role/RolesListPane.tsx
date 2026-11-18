import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { RoleListRow } from "@/components/role/RoleListRow";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { useSeesTeam } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { COMPARE_ROLES, NO_ROLES, type Role } from "@/models/Role";
import { roleHolders, roleless } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface RolesListPaneProps {
  roles: Role[];
  selectedId: string | undefined;
}

// The workspace's roles, the Owner first and outside the count as the page header counts them, each with how many hold it; then the members holding none, and every role side by side.
export const RolesListPane = ({ roles, selectedId }: RolesListPaneProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const seesTeam = useSeesTeam();
  const { data: team } = useFetchTeam(seesTeam);
  const [creating, setCreating] = useState(false);
  const owners = roles.filter((role) => role.is_owner_role);
  const others = roles.filter((role) => !role.is_owner_role);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader title="Roles" count={others.length} newLabel="New role" newPermission="roles:write" newVerb="create roles" onNew={() => setCreating(true)} />
      <div className="scroll-edge min-h-0 flex-1 space-y-4 overflow-y-auto pb-6 [scrollbar-gutter:stable]">
        <EnterList aria-label="Roles" className="pt-1">
          {[...owners, ...others].map((role) => (
            <RoleListRow
              key={role.id}
              id={role.id}
              name={role.name}
              line={role.description}
              owner={role.is_owner_role}
              selected={role.id === selectedId}
              count={team && roleHolders(team, workspaceId, role.id).length}
            />
          ))}
        </EnterList>
        <ListPaneGroup label="Overview">
          <RoleListRow id={COMPARE_ROLES} name="Compare roles" line="Every role side by side" selected={selectedId === COMPARE_ROLES} />
          {seesTeam && (
            <RoleListRow id={NO_ROLES} name="No roles" line="Members who see nothing yet" selected={selectedId === NO_ROLES} count={team && roleless(team, workspaceId).length} />
          )}
        </ListPaneGroup>
      </div>
      <RoleEditorDialog open={creating} onClose={() => setCreating(false)} />
    </div>
  );
};
