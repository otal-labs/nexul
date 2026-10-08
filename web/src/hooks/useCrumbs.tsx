import type { Crumb } from "@/components/PageBreadcrumb";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The selected workspace as the first crumb of a page; the slug stands in until the list loads.
export const useWorkspaceCrumb = (): Crumb => {
  const workspace = useSelectedWorkspace();
  const slug = useWorkspaceStore((s) => s.selectedWorkspaceSlug);
  const wsPath = useWorkspacePath();
  return { label: workspace?.name ?? slug, to: wsPath("/") };
};

// A project as a crumb that leads to its board.
export const useProjectCrumb = (projectId: string | undefined): Crumb | undefined => {
  const { data: project } = useFetchProject(projectId);
  const wsPath = useWorkspacePath();
  if (!project) return undefined;
  return { label: project.name, to: wsPath(`/board/${project.id}`) };
};
