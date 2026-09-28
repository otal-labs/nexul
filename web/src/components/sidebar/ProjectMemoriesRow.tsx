import { BrainIcon } from "lucide-react";

import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { isInterviewMemory, isWorkspaceMemory } from "@/models/Memory";
import { memoryPath, projectToken, type Project } from "@/models/Project";

interface ProjectMemoriesRowProps {
  project: Project;
  collapsed: boolean;
}

// Mirrors ProjectDocsRow exactly: each memory title is a direct sibling row, no intermediate "Memories" label.
// The fetch also returns workspace-scoped memories; this rail is the project's own, minus the interview's own row.
export const ProjectMemoriesRow = ({ project, collapsed }: ProjectMemoriesRowProps) => {
  const { data: memories = [] } = useFetchMemoriesByProject(project.id);
  const projectMemories = memories.filter((memory) => !isWorkspaceMemory(memory) && !isInterviewMemory(memory));

  return (
    <>
      {projectMemories.map((memory) => (
        <SidebarNavLink
          key={memory.id}
          to={memoryPath(projectToken(project), memory.id)}
          label={memory.title}
          icon={BrainIcon}
          collapsed={collapsed}
        />
      ))}
    </>
  );
};
