import { BrainIcon, ClipboardListIcon, FileTextIcon, LayoutDashboardIcon, SettingsIcon } from "lucide-react";

import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { boardPath, interviewPath, projectSettingsPath, projectToken, type Project } from "@/models/Project";

interface ProjectNavProps {
  project: Project;
  collapsed: boolean;
}

// Every row follows the viewer's workspace permissions; the Docs page itself lists only the docs they can open.
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
      {can?.("docs") && <SidebarNavLink to="/docs" label="Docs" icon={FileTextIcon} collapsed={collapsed} />}
      {can?.("memories") && <SidebarNavLink to="/memories" label="Memories" icon={BrainIcon} collapsed={collapsed} />}
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
