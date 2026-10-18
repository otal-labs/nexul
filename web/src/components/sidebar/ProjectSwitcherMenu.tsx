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
  collapsed: boolean;
}

export const ProjectSwitcherMenu = ({ projects, currentId, onSelect, onSettings, onPermissions, onCreate, collapsed }: ProjectSwitcherMenuProps) => (
  <SwitcherMenu createLabel="New project" onCreate={onCreate} collapsed={collapsed} createGate={{ permission: wizardDoorPermissions(true), verb: "create projects" }}>
    {projects.map((project) => (
      <ProjectSwitcherItem key={project.id} project={project} selected={project.id === currentId} onSelect={onSelect} onSettings={onSettings} onPermissions={onPermissions} />
    ))}
  </SwitcherMenu>
);
