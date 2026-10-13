import { ChevronRightIcon, XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { TargetMark } from "@/components/access/TargetMark";
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

// One role or person on the project: the whole row opens the panel on it, the X takes it off.
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
  const removeLabel = keptOut ? `Let ${target.name} back into ${project.name}` : `Remove ${target.name} from ${project.name}`;

  return (
    <li className="flex items-center gap-1">
      <button
        type="button"
        onClick={() => void openPanel(projectEntity(project), target.key)}
        className="-ml-2 flex min-w-0 flex-1 items-center gap-3 rounded-md py-2 pr-1 pl-2 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <TargetMark target={target} />
        <span className="min-w-0 flex-1">
          <span title={target.name} className="block truncate text-sm font-medium">
            {target.name}
          </span>
          <span className="block truncate text-xs text-muted-foreground">{line}</span>
        </span>
        <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </button>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="size-8 shrink-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            aria-label={removeLabel}
            onClick={() => void remove(target, grant, keptOut)}
          >
            <XIcon className="size-4" aria-hidden />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{removeLabel}</TooltipContent>
      </Tooltip>
    </li>
  );
};
