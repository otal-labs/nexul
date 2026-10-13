import { hasPermission, projectPermissions } from "@nexul/client-core/permissions";

import { useFetchExplanation } from "@/hooks/PermissionHooks";
import { useFetchMyRole, useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { PanelEntity } from "@/models/PermissionPanel";

// What managing each entity's rules takes there, when the server's explanation answers it.
const manageAction: Partial<Record<PanelEntity["type"], string>> = { doc: "permissions:write", folder: "permissions:write", stack: "stacks:write" };

// The server's manage check; a doc's, folder's or stack's costs a request, so call it from an open menu.
export const useCanManageAccess = (entity: Pick<PanelEntity, "type" | "id"> | undefined): boolean => {
  const canWritePlays = useHasPermission("plays:write");
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: mine } = useFetchMyRole(workspaceId);
  const manage = entity && manageAction[entity.type];
  const { data } = useFetchExplanation(manage ? entity.type : "doc", entity?.id ?? "", "", !!manage);
  if (entity?.type === "play") return canWritePlays;
  if (entity?.type === "project") return hasPermission(projectPermissions(mine, entity.id), "projects:access");
  return !!data?.effective.some((answer) => answer.action === manage && answer.allowed);
};
