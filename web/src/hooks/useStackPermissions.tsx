import type { RowMenu } from "@/components/listpane/rowMenu";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { stackEntity } from "@/models/PermissionPanel";

interface RuledStack {
  id: string;
  name: string;
  project_id: string;
  derived_from?: string | undefined;
}

// Permissions… for a project's stack and whether the viewer may open it; an instance stack or a branch deployment
// carries no rules of its own, so it asks nothing.
export const useStackPermissions = (stack: RuledStack): { menu: RowMenu; manages: boolean } => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const openPermissions = usePermissionsDialog();
  const entity = stackEntity(stack, workspaceId);
  const manages = useCanManageAccess(stack.project_id && !stack.derived_from ? entity : undefined);
  return { manages, menu: { permissions: manages ? { entity, onOpen: () => void openPermissions(entity) } : undefined } };
};
