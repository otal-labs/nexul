import { useSaveRules } from "@/hooks/PermissionHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { PanelTarget } from "@/hooks/usePanelTargets";
import { RuleState, type PermissionGrant } from "@/models/Permission";
import { IN_THIS_PROJECT } from "@/models/ProjectPeople";
import type { Project } from "@/models/Project";

// Listing a role or person on a project lets them in; taking them off drops every rule they hold there.
export const useProjectPeopleActions = (project: Project) => {
  const save = useSaveRules();
  const { open: confirm } = useConfirmationDialog();

  const add = (target: PanelTarget) =>
    save.mutate({
      resourceType: "project",
      resourceId: project.id,
      changes: [{ target: target.key, state: RuleState.Allow, actions: [IN_THIS_PROJECT] }],
      done: `${target.name} added to ${project.name}`,
    });

  const remove = async (target: PanelTarget, grant: PermissionGrant, keptOut: boolean) => {
    const ok = await confirm({
      title: keptOut ? `Let ${target.name} back into ${project.name}?` : `Remove ${target.name} from ${project.name}?`,
      message: keptOut
        ? "The rule keeping them out goes, with their other rules here, so their roles decide again."
        : `${target.kind === "role" ? "Its holders lose" : `${target.name} loses`} ${project.name} unless another role or a rule of their own lets them in. The rules here go too.`,
      confirmLabel: keptOut ? "Let back in" : "Remove",
    });
    if (!ok) return;
    save.mutate({
      resourceType: "project",
      resourceId: project.id,
      changes: [{ target: target.key, state: RuleState.Fallback, actions: [...grant.allow, ...grant.deny] }],
      done: keptOut ? `${target.name} is no longer kept out` : `${target.name} removed from ${project.name}`,
    });
  };

  return { add, remove, saving: save.isPending };
};
