import { TargetMark } from "@/components/access/TargetMark";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useProjectPeopleActions } from "@/hooks/useProjectPeopleActions";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import type { PermissionGrant } from "@/models/Permission";
import { grantKey, projectEntity } from "@/models/PermissionPanel";
import { holdItWord, rulesHere, rulesHereWord } from "@/models/ProjectPeople";
import type { Project } from "@/models/Project";
import { roleHolders } from "@/models/Team";

interface ProjectPeopleRowProps {
  project: Project;
  grant: PermissionGrant;
  keptOut?: boolean | undefined;
}

// One role or person on the project: the whole row opens the panel on it, its menu also takes it off.
export const ProjectPeopleRow = ({ project, grant, keptOut = false }: ProjectPeopleRowProps) => {
  const { targetOf } = usePanelTargets({ workspaceId: project.workspace_id });
  // Holder counts come from Team, so only a viewer who may read Team asks for it.
  const { data: team } = useFetchTeam(useCanOpenSection("team"));
  const openPanel = usePermissionsDialog();
  const { remove } = useProjectPeopleActions(project);
  const target = targetOf(grantKey(grant));
  const rules = rulesHere(grant);

  // Who holds a role, and what a person holds, only for a viewer the server shows the team to.
  const holders = team && target.kind === "role" ? roleHolders(team, project.workspace_id, target.id).length : undefined;
  const membership = team?.people.find((p) => p.id === target.id)?.workspaces.find((w) => w.workspace_id === project.workspace_id);
  const held = membership && (membership.roles?.map((r) => r.name) ?? [membership.role_name]).join(", ");

  const line = [
    holders !== undefined && holdItWord(holders),
    target.kind === "person" && (held || (membership && "no role")),
    keptOut && "kept out by name",
    !keptOut && rules > 0 && rulesHereWord(rules),
    !keptOut && rules === 0 && (target.kind === "role" ? "the role's own permissions" : "listed by name"),
  ]
    .filter(Boolean)
    .join(" · ");
  const open = () => void openPanel(projectEntity(project), target.key);

  return (
    <li className="group/row relative flex min-h-14 items-center gap-2 px-2 before:pointer-events-none before:absolute before:inset-x-0 before:inset-y-1 before:rounded-md before:transition-colors before:duration-150 before:ease-standard hover:before:bg-accent/40">
      <button type="button" onClick={open} className="relative flex min-w-0 flex-1 items-center gap-3 self-stretch text-left">
        <TargetMark target={target} />
        <span className="min-w-0 flex-1">
          <span title={target.name} className="block truncate text-sm font-medium">
            {target.name}
          </span>
          <span className="block truncate text-xs text-muted-foreground">{line}</span>
        </span>
      </button>
      <span className="relative flex shrink-0 [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:has-data-[state=open]:opacity-100">
        <RowActionsMenu
          subject={target.name}
          projectId={project.id}
          actions={[
            { label: "Open", onSelect: open },
            { label: keptOut ? "Let back in" : "Remove", destructive: !keptOut, onSelect: () => void remove(target, grant, keptOut) },
          ]}
        />
      </span>
    </li>
  );
};
