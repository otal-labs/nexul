import { PlusIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { navLinkClass, sectionLabelClass } from "@/components/SidebarNav";
import { ProjectNav } from "@/components/sidebar/ProjectNav";
import { ProjectSwitcher } from "@/components/sidebar/ProjectSwitcher";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { cn } from "@/lib/utils";

interface ProjectSectionProps {
  collapsed: boolean;
}

// One project at a time: a switcher picks it, and its pages are listed once below instead of once per project.
export const ProjectSection = ({ collapsed }: ProjectSectionProps) => {
  const { projects, current } = useSidebarProject();
  const navigate = useNavigate();

  return (
    <div className="flex flex-col gap-0.5">
      {!collapsed && <div className={sectionLabelClass}>Project</div>}
      {projects && projects.length === 0 && (
        <button
          type="button"
          onClick={() => void navigate("/wizard/project/project")}
          title={collapsed ? "New project" : undefined}
          className={cn(navLinkClass({ isActive: false }), "w-full", collapsed && "justify-center px-0")}
        >
          <span className="flex w-8 shrink-0 justify-center">
            <PlusIcon className="size-4" aria-hidden />
          </span>
          {!collapsed && <span className="flex-1 text-left">New project</span>}
        </button>
      )}
      {projects && current && <ProjectSwitcher projects={projects} current={current} collapsed={collapsed} />}
      {current && <ProjectNav project={current} collapsed={collapsed} />}
    </div>
  );
};
