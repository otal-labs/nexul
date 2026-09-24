import { BrainIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { isInterviewMemory, isWorkspaceMemory } from "@/models/Memory";
import { memoryPath, projectToken, type Project } from "@/models/Project";
import { cn } from "@/lib/utils";

interface ProjectMemoriesRowProps {
  project: Project;
  /** Icon-only, no title text — for the collapsed rail's expanded flyout. */
  collapsed?: boolean;
}

// Mirrors ProjectDocsRow exactly: each memory title is a direct sibling row, no intermediate "Memories" label.
// The fetch also returns workspace-scoped memories; this rail is the project's own, minus the interview's own row.
export const ProjectMemoriesRow = ({ project, collapsed = false }: ProjectMemoriesRowProps) => {
  const { data: memories = [] } = useFetchMemoriesByProject(project.id);
  const projectMemories = memories.filter((memory) => !isWorkspaceMemory(memory) && !isInterviewMemory(memory));

  return (
    <>
      {projectMemories.map((memory) => (
        <NavLink
          key={memory.id}
          to={memoryPath(projectToken(project), memory.id)}
          title={collapsed ? memory.title : undefined}
          className={({ isActive }) => cn(navLinkClass({ isActive }), collapsed && "justify-center")}
        >
          <BrainIcon className="size-4 shrink-0" aria-hidden />
          {!collapsed && <span className="truncate">{memory.title}</span>}
        </NavLink>
      ))}
    </>
  );
};
