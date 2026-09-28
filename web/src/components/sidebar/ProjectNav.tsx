import { ClipboardListIcon, LayoutDashboardIcon, SettingsIcon } from "lucide-react";

import { ProjectDocsRow } from "@/components/sidebar/ProjectDocsRow";
import { ProjectMemoriesRow } from "@/components/sidebar/ProjectMemoriesRow";
import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { boardPath, interviewPath, projectSettingsPath, projectToken, type Project } from "@/models/Project";

interface ProjectNavProps {
  project: Project;
  collapsed: boolean;
}

export const ProjectNav = ({ project, collapsed }: ProjectNavProps) => (
  <div className="flex flex-col gap-0.5">
    <SidebarNavLink to={boardPath(project)} label="Board" icon={LayoutDashboardIcon} collapsed={collapsed} />
    <SidebarNavLink
      to={interviewPath(projectToken(project))}
      label="Interview"
      icon={ClipboardListIcon}
      collapsed={collapsed}
    />
    <ProjectDocsRow project={project} collapsed={collapsed} />
    <ProjectMemoriesRow project={project} collapsed={collapsed} />
    <SidebarNavLink
      to={projectSettingsPath(projectToken(project))}
      label="Settings"
      icon={SettingsIcon}
      collapsed={collapsed}
    />
  </div>
);
