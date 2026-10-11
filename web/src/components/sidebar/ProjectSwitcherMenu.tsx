import { ProjectSwitcherItem } from "@/components/sidebar/ProjectSwitcherItem";
import { SwitcherMenu } from "@/components/SwitcherMenu";
import type { Project } from "@/models/Project";

interface ProjectSwitcherMenuProps {
  projects: Project[];
  currentId: string;
  onSelect: (project: Project) => void;
  onSettings: (project: Project) => void;
  // Absent for a viewer who may not create a project.
  onCreate?: (() => void) | undefined;
}

export const ProjectSwitcherMenu = ({ projects, currentId, onSelect, onSettings, onCreate }: ProjectSwitcherMenuProps) => (
  <SwitcherMenu createLabel="New project" onCreate={onCreate}>
    {projects.map((project) => (
      <ProjectSwitcherItem key={project.id} project={project} selected={project.id === currentId} onSelect={onSelect} onSettings={onSettings} />
    ))}
  </SwitcherMenu>
);
