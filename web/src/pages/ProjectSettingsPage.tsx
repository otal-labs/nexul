import { Link, useParams, useSearchParams } from "react-router";

import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import {
  DEFAULT_PROJECT_SETTINGS_SECTION,
  isProjectSettingsSection,
} from "@/components/settings/ProjectSettingsNav";
import { ProjectSettingsContent } from "@/components/settings/ProjectSettingsContent";
import { Button } from "@/components/ui/button";
import { useFetchProject, useFetchProjects } from "@/hooks/ProjectHooks";
import { resolveProject } from "@/models/Project";

export const ProjectSettingsPage = () => {
  // The URL token is the project's prefix or (old links) its id, same resolution rule as the board route.
  const { projectId: routeParam = "" } = useParams();
  const { data: projects = [], isPending, error } = useFetchProjects();
  const resolved = resolveProject(projects, routeParam);
  const projectId = resolved?.id ?? "";
  const notFound = !isPending && !error && !resolved;
  const { data: project } = useFetchProject(projectId);
  const [searchParams] = useSearchParams();
  const rawSection = searchParams.get("section");
  const section = isProjectSettingsSection(rawSection) ? rawSection : DEFAULT_PROJECT_SETTINGS_SECTION;

  return (
    <Container className="mx-auto max-w-5xl py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {notFound && (
        <EmptyState
          title="Project not found"
          action={
            <Button asChild size="sm">
              <Link to="/board">Go to your board</Link>
            </Button>
          }
        />
      )}
      {project && <ProjectSettingsContent project={project} section={section} />}
    </Container>
  );
};
