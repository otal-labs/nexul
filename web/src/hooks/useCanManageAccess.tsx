import { hasPermission, projectPermissions } from "@nexul/client-core/permissions";

import { useFetchExplanation } from "@/hooks/PermissionHooks";
import { useFetchMyRole, useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { PanelEntity } from "@/models/PermissionPanel";

// The server's check (a doc's permissions:write, a play's plays:write, a project's projects:access there); a doc's
// costs a request, so call it from an open menu.
export const useCanManageAccess = (entity: Pick<PanelEntity, "type" | "id"> | undefined): boolean => {
  const canWritePlays = useHasPermission("plays:write");
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: mine } = useFetchMyRole(workspaceId);
  const { data } = useFetchExplanation("doc", entity?.id ?? "", "", entity?.type === "doc");
  if (entity?.type === "play") return canWritePlays;
  if (entity?.type === "project") return hasPermission(projectPermissions(mine, entity.id), "projects:access");
  return !!data?.effective.some((answer) => answer.action === "permissions:write" && answer.allowed);
};
