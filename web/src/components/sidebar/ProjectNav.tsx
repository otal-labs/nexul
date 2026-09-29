import { ClipboardListIcon, LayoutDashboardIcon, SettingsIcon } from "lucide-react";

import { ProjectDocsRow } from "@/components/sidebar/ProjectDocsRow";
import { ProjectMemoriesRow } from "@/components/sidebar/ProjectMemoriesRow";
import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { boardPath, interviewPath, projectSettingsPath, projectToken, type Project } from "@/models/Project";

interface ProjectNavProps {
  project: Project;
  collapsed: boolean;
}

// Docs are listed as the server filters them per doc; every other row follows the viewer's workspace permissions.
export const ProjectNav = ({ project, collapsed }: ProjectNavProps) => {
  const can = useAreaAccess();
  return (
    <div className="flex flex-col gap-0.5">
      {can?.("tickets") && (
        <SidebarNavLink to={boardPath(project)} label="Board" icon={LayoutDashboardIcon} collapsed={collapsed} />
      )}
      {can?.("memories") && (
        <SidebarNavLink
          to={interviewPath(projectToken(project))}
          label="Interview"
          icon={ClipboardListIcon}
          collapsed={collapsed}
        />
      )}
      <ProjectDocsRow project={project} collapsed={collapsed} />
      {can?.("memories") && <ProjectMemoriesRow project={project} collapsed={collapsed} />}
      {can?.("projects") && (
        <SidebarNavLink
          to={projectSettingsPath(projectToken(project))}
          label="Settings"
          icon={SettingsIcon}
          collapsed={collapsed}
        />
      )}
    </div>
  );
};
