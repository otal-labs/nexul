import { useMemo } from "react";

import { MemoryProjectSection } from "@/components/memory/MemoryProjectSection";
import { isWorkspaceMemory, type Memory } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";

interface MemoriesFeedProps {
  memories: Memory[];
  projects: Project[];
}

interface ProjectGroup {
  key: string;
  projectName: string;
  projectToken: string;
  memories: Memory[];
}

// Groups the flat, API-returned list into a workspace-wide section first, then one section per project.
const groupMemories = (memories: Memory[], projects: Project[]): ProjectGroup[] => {
  const workspaceMemories = memories.filter(isWorkspaceMemory);
  const byProject = new Map<string, Memory[]>();
  for (const memory of memories) {
    if (isWorkspaceMemory(memory)) continue;
    const group = byProject.get(memory.project_id) ?? [];
    group.push(memory);
    byProject.set(memory.project_id, group);
  }
  const workspaceGroup: ProjectGroup[] =
    workspaceMemories.length > 0
      ? [{ key: "workspace", projectName: "Workspace", projectToken: "", memories: workspaceMemories }]
      : [];
  const projectGroups = projects
    .filter((project) => byProject.has(project.id))
    .map((project) => ({
      key: project.id,
      projectName: project.name,
      projectToken: projectToken(project),
      memories: byProject.get(project.id) ?? [],
    }));
  return [...workspaceGroup, ...projectGroups];
};

export const MemoriesFeed = ({ memories, projects }: MemoriesFeedProps) => {
  const groups = useMemo(() => groupMemories(memories, projects), [memories, projects]);

  return (
    <div>
      {groups.map((group) => (
        <MemoryProjectSection
          key={group.key}
          projectName={group.projectName}
          projectToken={group.projectToken}
          memories={group.memories}
        />
      ))}
    </div>
  );
};
