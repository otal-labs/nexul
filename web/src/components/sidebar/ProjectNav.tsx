import { BrainIcon, ClipboardListIcon, FileTextIcon, LayoutDashboardIcon, SettingsIcon } from "lucide-react";

import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath, interviewPath, projectSettingsPath, projectToken, type Project } from "@/models/Project";

interface ProjectNavProps {
  project: Project;
  collapsed: boolean;
}

// Every row follows the viewer's workspace permissions; the Docs page itself lists only the docs they can open.
export const ProjectNav = ({ project, collapsed }: ProjectNavProps) => {
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  return (
    <div className="flex flex-col gap-0.5">
      {can?.("tickets") && (
        <SidebarNavLink to={wsPath(boardPath(project))} label="Board" icon={LayoutDashboardIcon} collapsed={collapsed} />
      )}
      {can?.("memories") && (
        <SidebarNavLink
          to={wsPath(interviewPath(projectToken(project)))}
          label="Interview"
          icon={ClipboardListIcon}
          collapsed={collapsed}
        />
      )}
      {can?.("docs") && <SidebarNavLink to={wsPath("/docs")} label="Docs" icon={FileTextIcon} collapsed={collapsed} />}
      {can?.("memories") && <SidebarNavLink to={wsPath("/memories")} label="Memories" icon={BrainIcon} collapsed={collapsed} />}
      {can?.("projects") && (
        <SidebarNavLink
          to={wsPath(projectSettingsPath(projectToken(project)))}
          label="Settings"
          icon={SettingsIcon}
          collapsed={collapsed}
        />
      )}
    </div>
  );
};
