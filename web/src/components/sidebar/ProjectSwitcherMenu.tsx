import { ProjectSwitcherItem } from "@/components/sidebar/ProjectSwitcherItem";
import { SwitcherMenu } from "@/components/SwitcherMenu";
import { wizardDoorPermissions } from "@/models/Access";
import type { Project } from "@/models/Project";

interface ProjectSwitcherMenuProps {
  projects: Project[];
  currentId: string;
  onSelect: (project: Project) => void;
  onSettings: (project: Project) => void;
  onPermissions: (project: Project) => void;
  onCreate: () => void;
}

export const ProjectSwitcherMenu = ({ projects, currentId, onSelect, onSettings, onPermissions, onCreate }: ProjectSwitcherMenuProps) => (
  <SwitcherMenu createLabel="New project" onCreate={onCreate} createGate={{ permission: wizardDoorPermissions(true), verb: "create projects" }}>
    {projects.map((project) => (
      <ProjectSwitcherItem key={project.id} project={project} selected={project.id === currentId} onSelect={onSelect} onSettings={onSettings} onPermissions={onPermissions} />
    ))}
  </SwitcherMenu>
);
