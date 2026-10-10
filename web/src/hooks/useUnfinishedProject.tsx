import { useFetchProjectRepos, useFetchProjects } from "@/hooks/ProjectHooks";
import { useUnfinishedProjectStore } from "@/stores/unfinishedProjectStore";
import { RepoRole } from "@/enums/Project";
import type { Project } from "@/models/Project";

// The project this browser named in the wizard while it still has no repository to deploy, in this workspace.
export const useUnfinishedProject = (): Project | undefined => {
  const projectId = useUnfinishedProjectStore((s) => s.projectId);
  const { data: projects } = useFetchProjects(!!projectId);
  const project = projects?.find((p) => p.id === projectId);
  const { data: repos } = useFetchProjectRepos(project?.id);
  if (!project || !repos || repos.some((r) => r.role === RepoRole.App)) return undefined;
  return project;
};
