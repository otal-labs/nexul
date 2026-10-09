import { Link, useParams } from "react-router";

import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectInterview } from "@/components/memory/ProjectInterview";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { resolveProject } from "@/models/Project";

export const InterviewPage = () => {
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const wsPath = useWorkspacePath();
  const { projectId: routeParam = "" } = useParams();
  const { data: projects, isPending, error } = useFetchProjects();
  const project = projects && resolveProject(projects, routeParam);

  return (
    <Container className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && !project && (
        <EmptyState
          title="Project not found"
          action={
            canOpenBoard && (
              <Button asChild size="sm">
                <Link to={wsPath("/board")}>Go to your board</Link>
              </Button>
            )
          }
        />
      )}
      {project && <ProjectInterview project={project} />}
    </Container>
  );
};
