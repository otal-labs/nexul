import { Link, useParams } from "react-router";

import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectInterview } from "@/components/memory/ProjectInterview";
import { Button } from "@/components/ui/button";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { resolveProject } from "@/models/Project";

export const InterviewPage = () => {
  const { projectId: routeParam = "" } = useParams();
  const { data: projects, isPending, error } = useFetchProjects();
  const project = projects && resolveProject(projects, routeParam);

  return (
    <Container className="p-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && !project && (
        <EmptyState
          title="Project not found"
          action={
            <Button asChild size="sm">
              <Link to="/board">Go to your board</Link>
            </Button>
          }
        />
      )}
      {project && <ProjectInterview project={project} />}
    </Container>
  );
};
