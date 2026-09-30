import { PlusIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { navLinkClass, sectionLabelClass } from "@/components/SidebarNav";
import { ProjectNav } from "@/components/sidebar/ProjectNav";
import { ProjectSwitcher } from "@/components/sidebar/ProjectSwitcher";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { NEW_PROJECT_PATH } from "@/models/Project";
import { cn } from "@/lib/utils";

interface ProjectSectionProps {
  collapsed: boolean;
}

// One project at a time: a switcher picks it, and its pages are listed once below instead of once per project.
export const ProjectSection = ({ collapsed }: ProjectSectionProps) => {
  const { projects, current } = useSidebarProject();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const can = useAreaAccess();
  const canCreate = can?.("newProject") ?? false;
  // The switcher only earns its place when the viewer can open something inside a project.
  const readsProjects = !!can && (["projects", "tickets", "memories", "docs"] as const).some((area) => can(area));
  const offerCreate = !!projects && projects.length === 0 && canCreate;
  const showSwitcher = readsProjects && !!current;

  if (!offerCreate && !showSwitcher) return null;
  return (
    <div className="flex flex-col gap-0.5">
      {!collapsed && <div className={sectionLabelClass}>Project</div>}
      {offerCreate && (
        <button
          type="button"
          onClick={() => void navigate(wsPath(NEW_PROJECT_PATH))}
          title={collapsed ? "New project" : undefined}
          className={cn(navLinkClass({ isActive: false }), "w-full", collapsed && "justify-center px-0")}
        >
          <span className="flex w-8 shrink-0 justify-center">
            <PlusIcon className="size-4" aria-hidden />
          </span>
          {!collapsed && <span className="flex-1 text-left">New project</span>}
        </button>
      )}
      {projects && current && showSwitcher && (
        <ProjectSwitcher projects={projects} current={current} collapsed={collapsed} />
      )}
      {current && showSwitcher && <ProjectNav project={current} collapsed={collapsed} />}
    </div>
  );
};
