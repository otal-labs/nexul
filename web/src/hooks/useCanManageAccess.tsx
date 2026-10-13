import { hasPermission, projectPermissions } from "@nexul/client-core/permissions";

import { useFetchExplanation } from "@/hooks/PermissionHooks";
import { useFetchMyRole, useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { PanelEntity } from "@/models/PermissionPanel";

// The server's manage check; a doc's or folder's costs a request, so call it from an open menu.
export const useCanManageAccess = (entity: Pick<PanelEntity, "type" | "id"> | undefined): boolean => {
  const canWritePlays = useHasPermission("plays:write");
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: mine } = useFetchMyRole(workspaceId);
  const shared = entity?.type === "doc" || entity?.type === "folder";
  const { data } = useFetchExplanation(shared ? entity.type : "doc", entity?.id ?? "", "", shared);
  if (entity?.type === "play") return canWritePlays;
  if (entity?.type === "project") return hasPermission(projectPermissions(mine, entity.id), "projects:access");
  return !!data?.effective.some((answer) => answer.action === "permissions:write" && answer.allowed);
};
