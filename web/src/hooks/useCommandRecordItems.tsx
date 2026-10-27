import { BoxesIcon, BrainIcon, WorkflowIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { useCanOpen } from "@/hooks/AccessHooks";
import { useFetchAutomations } from "@/hooks/AutomationHooks";
import { useFetchMemories } from "@/hooks/MemoryHooks";
import { useFetchStacks } from "@/hooks/StackHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { memoryPath, projectTokenById } from "@/models/Project";

// Typed, the workspace's memories, stacks and automations by name, from the lists their pages read; each is fetched
// only once something is typed and only when the viewer may open its page.
export const useCommandRecordItems = (browsing: boolean): CommandGroup[] => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const canOpen = useCanOpen();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { projects } = useSidebarProject();
  const { data: memories } = useFetchMemories(browsing ? "" : workspaceId);
  const { data: stacks } = useFetchStacks(undefined, !browsing && canOpen("stacks") === true);
  const { data: automations } = useFetchAutomations(!browsing && canOpen("automations") === true);
  const go = (path: string) => () => void navigate(wsPath(path));
  if (browsing) return [];

  const inWorkspace = new Set((projects ?? []).map((project) => project.id));
  const memoryItems: CommandItem[] = (memories ?? []).map((memory) => ({
    id: `memory:${memory.id}`,
    label: memory.title,
    icon: BrainIcon,
    keywords: "memory",
    run: go(memoryPath(projectTokenById(projects ?? [], memory.project_id), memory.id)),
  }));
  const stackItems: CommandItem[] = (stacks ?? [])
    .filter((stack) => inWorkspace.has(stack.project_id))
    .map((stack) => ({ id: `stack:${stack.id}`, label: stack.name, icon: BoxesIcon, keywords: "stack", run: go(`/stacks/${stack.id}`) }));
  const automationItems: CommandItem[] = (automations ?? []).map((automation) => ({
    id: `automation:${automation.id}`,
    label: automation.name,
    icon: WorkflowIcon,
    keywords: "automation",
    run: go(`/automations/${automation.id}`),
  }));

  return [
    { heading: "Memories", items: memoryItems },
    { heading: "Stacks", items: stackItems },
    { heading: "Automations", items: automationItems },
  ];
};
