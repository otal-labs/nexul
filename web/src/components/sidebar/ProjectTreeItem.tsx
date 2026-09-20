import { ChevronDownIcon, LayoutDashboardIcon, PlusIcon, SettingsIcon } from "lucide-react";
import { NavLink } from "react-router";

import { ProjectCollapsiblePanel } from "@/components/project/ProjectCollapsiblePanel";
import { navLinkClass } from "@/components/SidebarNav";
import { ProjectDocsRow } from "@/components/sidebar/ProjectDocsRow";
import { ProjectMemoriesRow } from "@/components/sidebar/ProjectMemoriesRow";
import { cn } from "@/lib/utils";
import { boardPath, projectSettingsPath, projectToken, type Project } from "@/models/Project";

interface ProjectTreeItemProps {
  project: Project;
  collapsed: boolean;
  open: boolean;
  onToggle: () => void;
  onCreateDoc: () => void;
}

// The prefix badge is the leading icon itself (no folder icon); collapsed rows need explicit `justify-center`.
export const ProjectTreeItem = ({
  project,
  collapsed,
  open,
  onToggle,
  onCreateDoc,
}: ProjectTreeItemProps) => {
  return (
    <>
      {collapsed && (
        <div className="flex flex-col gap-0.5">
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={open}
            title={project.name}
            className={cn(navLinkClass({ isActive: false }), "w-full justify-center")}
          >
            <span className="font-mono text-[10px] font-semibold">
              {project.prefix || project.name[0]?.toUpperCase()}
            </span>
          </button>
          {open && (
            <div className="flex flex-col gap-0.5">
              <NavLink
                to={boardPath(project)}
                className={({ isActive }) => cn(navLinkClass({ isActive }), "justify-center")}
                title="Board"
              >
                <LayoutDashboardIcon className="size-4 shrink-0" aria-hidden />
              </NavLink>
              <ProjectDocsRow project={project} collapsed />
              <ProjectMemoriesRow project={project} collapsed />
              <NavLink
                to={projectSettingsPath(projectToken(project))}
                className={({ isActive }) => cn(navLinkClass({ isActive }), "justify-center")}
                title="Settings"
              >
                <SettingsIcon className="size-4 shrink-0" aria-hidden />
              </NavLink>
            </div>
          )}
        </div>
      )}
      {!collapsed && (
        <div className="flex flex-col">
          <div className="group/project flex items-center">
            <button
              type="button"
              onClick={onToggle}
              aria-expanded={open}
              className={cn(navLinkClass({ isActive: false }), "w-full flex-1")}
            >
              {project.prefix !== "" && (
                <span className="w-8 shrink-0 text-center font-mono text-[11px] font-semibold text-muted-foreground">
                  {project.prefix}
                </span>
              )}
              <span className="flex-1 truncate text-left">{project.name}</span>
              <ChevronDownIcon
                className={cn(
                  "size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard",
                  open && "rotate-180",
                )}
                aria-hidden
              />
            </button>
            <button
              type="button"
              onClick={onCreateDoc}
              aria-label={`New doc in ${project.name}`}
              className="shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard hover:bg-accent/60 hover:text-foreground group-hover/project:opacity-100 focus-visible:opacity-100"
            >
              <PlusIcon className="size-3.5" aria-hidden />
            </button>
          </div>
          <ProjectCollapsiblePanel open={open}>
            <div className="ml-[26px] flex flex-col gap-0.5 border-l border-border/70 py-0.5">
              <NavLink to={boardPath(project)} className={navLinkClass}>
                <LayoutDashboardIcon className="size-4 shrink-0" aria-hidden />
                <span className="truncate">Board</span>
              </NavLink>
              <ProjectDocsRow project={project} />
              <ProjectMemoriesRow project={project} />
              <NavLink to={projectSettingsPath(projectToken(project))} className={navLinkClass}>
                <SettingsIcon className="size-4 shrink-0" aria-hidden />
                <span className="truncate">Settings</span>
              </NavLink>
            </div>
          </ProjectCollapsiblePanel>
        </div>
      )}
    </>
  );
};
