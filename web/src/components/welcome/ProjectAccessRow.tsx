import { Link } from "react-router";

import { ProjectMark } from "@/components/project/ProjectMark";
import { Button } from "@/components/ui/button";
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

// One project the person is in: its mark, name and level, then what that level lets them do there.
export const ProjectAccessRow = ({ project, workspaceSlug, domains, held }: ProjectAccessRowProps) => (
  <li className="flex items-center gap-3 py-3">
    <ProjectMark project={project} className="size-8 rounded-md text-[11px]" />
    <div className="min-w-0 flex-1">
      <p className="flex items-baseline gap-2 text-sm font-medium">
        <span className="truncate">{project.name}</span>
        <span className="shrink-0 font-mono text-xs font-normal text-muted-foreground">{projectLevelLabel(domains, held)}</span>
      </p>
      <p className="truncate text-xs text-muted-foreground">{accessSummary(domains, held)}</p>
    </div>
    <Button asChild variant="ghost" size="sm" className="shrink-0">
      <Link to={workspacePath(workspaceSlug, boardPath(project))}>Open</Link>
    </Button>
  </li>
);
