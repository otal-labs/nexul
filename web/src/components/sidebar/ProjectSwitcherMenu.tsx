import { ProjectMark } from "@/components/project/ProjectMark";
import { SwitcherMenu, SwitcherMenuItem } from "@/components/SwitcherMenu";
import { projectTile, type Project } from "@/models/Project";

interface ProjectSwitcherMenuProps {
  projects: Project[];
  currentId: string;
  onSelect: (project: Project) => void;
  // Absent for a viewer who may not create a project.
  onCreate?: (() => void) | undefined;
}

export const ProjectSwitcherMenu = ({ projects, currentId, onSelect, onCreate }: ProjectSwitcherMenuProps) => (
  <SwitcherMenu createLabel="New project" onCreate={onCreate}>
    {projects.map((project) => (
      <SwitcherMenuItem
        key={project.id}
        tile={projectTile(project)}
        mark={<ProjectMark project={project} className="h-5 w-auto min-w-6 rounded-sm px-1 text-[10px] shadow-none" />}
        name={project.name}
        selected={project.id === currentId}
        onSelect={() => onSelect(project)}
      />
    ))}
  </SwitcherMenu>
);
