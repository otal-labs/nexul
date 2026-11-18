import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
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

// The workspace's roles, the Owner first, each with how many hold it;
// under them, pinned outside the scroller so they stay in view, every role side by side and the members holding none.
export const RolesListPane = ({ roles, selectedId }: RolesListPaneProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const seesTeam = useSeesTeam();
  const { data: team } = useFetchTeam(seesTeam);
  const [creating, setCreating] = useState(false);
  const [search, setSearch] = useState("");
  const needle = search.trim().toLowerCase();
  const matches = (role: Role) => !needle || `${role.name} ${role.description}`.toLowerCase().includes(needle);
  const owners = roles.filter((role) => role.is_owner_role);
  const others = roles.filter((role) => !role.is_owner_role);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader title="Roles" count={roles.length} newLabel="New role" newPermission="roles:write" newVerb="create roles" onNew={() => setCreating(true)} search={search} onSearch={setSearch} />
      <div className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-2 [scrollbar-gutter:stable_both-edges]">
        <EnterList aria-label="Roles">
          {[...owners, ...others].filter(matches).map((role) => (
            <RoleListRow
              key={role.id}
              id={role.id}
              name={role.name}
              line={role.description || (role.is_owner_role ? "Holds everything here" : "")}
              owner={role.is_owner_role}
              selected={role.id === selectedId}
              count={team && roleHolders(team, workspaceId, role.id).length}
            />
          ))}
        </EnterList>
        {needle && ![...owners, ...others].some(matches) && <ListPaneNoMatch onClear={() => setSearch("")} />}
      </div>
      <div className="shrink-0 overflow-y-hidden border-t border-border pt-3 [scrollbar-gutter:stable_both-edges]">
        <EnterList aria-label="Overview">
          <RoleListRow id={COMPARE_ROLES} name="Compare roles" line="" dense selected={selectedId === COMPARE_ROLES} />
          {seesTeam && (
            <RoleListRow id={NO_ROLES} name="No roles" line="" dense selected={selectedId === NO_ROLES} count={team && roleless(team, workspaceId).length} />
          )}
        </EnterList>
      </div>
      <RoleEditorDialog open={creating} onClose={() => setCreating(false)} />
    </div>
  );
};
