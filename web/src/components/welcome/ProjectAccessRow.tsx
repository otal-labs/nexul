import { ChevronRight } from "lucide-react";
import { Link } from "react-router";

import { ProjectMark } from "@/components/project/ProjectMark";
import { wizardCardClass } from "@/components/wizard/WizardCardList";
import { accessSummary, projectLevelLabel, type PermissionDomain } from "@/models/PermissionLevel";
import { boardPath, type Project } from "@/models/Project";
import { workspacePath } from "@/models/Workspace";

interface ProjectAccessRowProps {
  project: Project;
  workspaceSlug: string;
  domains: PermissionDomain[];
  // What the person holds inside this project.
  held: string[];
}

// One project the person is in, as a card that opens its board: mark, name, its level as a chip, the areas only when they differ.
export const ProjectAccessRow = ({ project, workspaceSlug, domains, held }: ProjectAccessRowProps) => {
  const level = projectLevelLabel(domains, held);
  return (
    <li className={wizardCardClass}>
      <Link to={workspacePath(workspaceSlug, boardPath(project))} className="group inset-focus flex items-center gap-3 rounded-lg px-4 py-3">
        <ProjectMark project={project} className="size-8 rounded-md text-[11px]" />
        <span className="min-w-0 flex-1">
          <span className="line-clamp-2 text-sm font-medium break-words">{project.name}</span>
          {level === "Custom" && <span className="block truncate text-xs text-muted-foreground">{accessSummary(domains, held)}</span>}
        </span>
        <span className="shrink-0 rounded border border-border px-1.5 py-px font-mono text-[11px] leading-4 text-muted-foreground">{level}</span>
        <ChevronRight className="size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:translate-x-0.5 motion-reduce:transition-none" aria-hidden />
      </Link>
    </li>
  );
};
