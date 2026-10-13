import { NoRolesChip, RoleChip } from "@/components/role/RoleChip";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";

// The workspace's /me, which the switcher and sidebar already keep cached and live.
export const YourRolesRow = ({ workspace }: { workspace: Workspace }) => {
  const { data: me } = useFetchMyRole(workspace.id);
  const roles = me?.roles;
  return (
    <li className="flex min-h-11 items-center gap-3 py-2">
      <span className="min-w-0 flex-1 truncate text-sm font-medium">{workspace.name}</span>
      <span className="flex flex-wrap justify-end gap-1">
        {roles && roles.length === 0 && <NoRolesChip />}
        {roles?.map((role) => (
          <RoleChip key={role.id} name={role.name} />
        ))}
      </span>
    </li>
  );
};
