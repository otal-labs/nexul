import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

// The default workspace's role a first sign-in gives, by name.
export const SsoJoinRole = ({ roleId }: { roleId: string }) => {
  const { data: roles } = useFetchWorkspaceRoles(INSTANCE_WORKSPACE_ID);
  return <span className="truncate text-sm">{roles?.find((r) => r.id === roleId)?.name ?? "A role that no longer exists"}</span>;
};
