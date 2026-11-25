import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import { useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AGENT_PATH } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AgentProjectPaneProps {
  projectId: string;
}

// Which of your computers a project's runs use, in order, and the checkout on each. Waits for the links too, so the
// form never seeds "Uses your defaults" before the project's own link has loaded.
export const AgentProjectPane = ({ projectId }: AgentProjectPaneProps) => {
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchWorkspaceProjects(workspaceId);
  const { data: links, error: linksError, isPending: linksPending } = useFetchProjectLinks();
  const project = projects?.find((p) => p.id === projectId);
  const link = links?.find((l) => l.project_id === projectId);
  const summary = useProjectLinkSummary(link);
  const error = projectsError || linksError;

  return (
    <Container className="space-y-6 py-8 sm:pr-3.5">
      {(projectsPending || linksPending) && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && links && !project && <NoDataDisplay message="That project isn't here" />}
      {project && links && (
        <PageHeader
          crumbs={[{ label: "Agent", to: wsPath(AGENT_PATH) }, { label: "Projects" }]}
          leading={<ProjectMark project={project} className="size-9 rounded-md text-xs shadow-none" />}
          title={project.name}
          meta={summary}
        />
      )}
      {project && links && (
        <div className="@container max-w-2xl">
          <ProjectLinkForm projectId={project.id} projectName={project.name} link={link ?? { project_id: project.id, computers: [], checkouts: [] }} />
        </div>
      )}
    </Container>
  );
};
