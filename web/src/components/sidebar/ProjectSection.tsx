import { PlusIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { navLinkClass, sectionLabelClass } from "@/components/SidebarNav";
import { ProjectNav } from "@/components/sidebar/ProjectNav";
import { ProjectSetupNav } from "@/components/sidebar/ProjectSetupNav";
import { ProjectSwitcher } from "@/components/sidebar/ProjectSwitcher";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { useAnyProjectAreaAccess } from "@/hooks/AccessHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { inSetup, NEW_PROJECT_PATH } from "@/models/Project";
import { cn } from "@/lib/utils";

interface ProjectSectionProps {
  collapsed: boolean;
}

// One project at a time: a switcher picks it, and its pages are listed once below instead of once per project.
export const ProjectSection = ({ collapsed }: ProjectSectionProps) => {
  const { projects, current } = useSidebarProject();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const can = useAnyProjectAreaAccess();
  const canCreate = can?.("newProject") ?? false;
  // The switcher only earns its place when the viewer can open something inside some project.
  const readsProjects = !!can && (["projects", "tickets", "memories", "docs"] as const).some((area) => can(area));
  const offerCreate = !!projects && projects.length === 0 && canCreate;
  const showSwitcher = readsProjects && !!current;

  if (!offerCreate && !showSwitcher) return null;
  return (
    <div className="flex flex-col gap-0.5">
      {!collapsed && <div className={sectionLabelClass}>Project</div>}
      {collapsed && <div className="mx-2 my-2 border-t border-border" aria-hidden />}
      {offerCreate && (
        <RailTooltip label="New project" collapsed={collapsed}>
          <button
            type="button"
            onClick={() => void navigate(wsPath(NEW_PROJECT_PATH))}
            aria-label={collapsed ? "New project" : undefined}
            className={cn(navLinkClass({ isActive: false }), "w-full")}
          >
            <span className="flex w-8 shrink-0 justify-center">
              <PlusIcon className="size-4" aria-hidden />
            </span>
            {!collapsed && <span className="flex-1 text-left">New project</span>}
          </button>
        </RailTooltip>
      )}
      {projects && current && showSwitcher && (
        <ProjectSwitcher projects={projects} current={current} collapsed={collapsed} />
      )}
      {current && showSwitcher && !inSetup(current) && <ProjectNav project={current} collapsed={collapsed} />}
      {current && showSwitcher && inSetup(current) && <ProjectSetupNav project={current} collapsed={collapsed} />}
    </div>
  );
};
