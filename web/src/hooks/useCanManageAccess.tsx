import { useFetchExplanation } from "@/hooks/PermissionHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { PanelEntity } from "@/models/PermissionPanel";

// The server's check (a doc's permissions:write, a play's plays:write); a doc's costs a request, so call it from an open menu.
export const useCanManageAccess = (entity: Pick<PanelEntity, "type" | "id"> | undefined): boolean => {
  const canWritePlays = useHasPermission("plays:write");
  const { data } = useFetchExplanation("doc", entity?.id ?? "", "", entity?.type === "doc");
  if (entity?.type === "play") return canWritePlays;
  return !!data?.effective.some((answer) => answer.action === "permissions:write" && answer.allowed);
};
