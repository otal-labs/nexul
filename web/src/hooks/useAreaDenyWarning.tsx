import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import type { PermissionAnswer } from "@/models/Permission";
import { targetScope } from "@/models/PermissionPanel";

// Before a Deny on an instance area's read that the target holds today: a rule beats every role, so it is their last way in.
export const useAreaDenyWarning = () => {
  const entity = usePanelEntity();
  const { open } = useConfirmationDialog();
  const { targetOf } = usePanelTargets(targetScope(entity));

  return async (targetKey: string, action: string, label: string, answer: PermissionAnswer | undefined) => {
    if (entity.type !== "instance_area" || !action.endsWith(":read") || !answer?.allowed) return true;
    const target = targetOf(targetKey);
    const message =
      target.kind === "person"
        ? `${target.name} can open ${entity.name} today. A Deny for them beats every role they hold in every workspace, so it takes away their last way in.`
        : `Everyone holding ${target.name} can open ${entity.name} today. A Deny for the role takes it from each of them, unless another of their roles has an Allow here or a rule names them.`;
    return open({ title: `Deny ${label} to ${target.name}?`, message, confirmLabel: "Deny" });
  };
};
