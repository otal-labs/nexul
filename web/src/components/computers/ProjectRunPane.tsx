import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PaneHeader } from "@/components/computers/PaneHeader";
import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import { useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface ProjectRunPaneProps {
  projectId: string;
}

// Waits for the links too, so the form never seeds "Uses your defaults" before the project's own link has loaded.
export const ProjectRunPane = ({ projectId }: ProjectRunPaneProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchWorkspaceProjects(workspaceId);
  const { data: links, error: linksError, isPending: linksPending } = useFetchProjectLinks();
  const project = projects?.find((p) => p.id === projectId);
  const link = links?.find((l) => l.project_id === projectId);
  const summary = useProjectLinkSummary(link);
  const error = projectsError || linksError;

  return (
    <div className="space-y-6">
      {(projectsPending || linksPending) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && links && !project && <NoDataDisplay message="That project isn't here" />}
      {project && links && <PaneHeader title={project.name} meta={summary} />}
      {project && links && (
        <div className="@container max-w-2xl">
          <ProjectLinkForm projectId={project.id} projectName={project.name} link={link ?? { project_id: project.id, computers: [], checkouts: [] }} />
        </div>
      )}
    </div>
  );
};
