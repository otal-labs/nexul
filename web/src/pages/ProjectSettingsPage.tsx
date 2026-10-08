import { Link, useParams } from "react-router";

import { Container } from "@/components/Container";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import type { Crumb } from "@/components/PageBreadcrumb";
import { PageHeader } from "@/components/PageHeader";
import {
  DEFAULT_PROJECT_SETTINGS_SECTION,
  isProjectSettingsSection,
} from "@/components/settings/ProjectSettingsNav";
import { ProjectSettingsContent } from "@/components/settings/ProjectSettingsContent";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProject, useFetchProjects } from "@/hooks/ProjectHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { resolveProject } from "@/models/Project";

export const ProjectSettingsPage = () => {
  // The URL token is the project's prefix, or its id when it has none; the board route resolves it the same way.
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const wsPath = useWorkspacePath();
  const { projectId: routeParam = "", section: rawSection } = useParams();
  const { data: projects = [], isPending, error } = useFetchProjects();
  const resolved = resolveProject(projects, routeParam);
  const projectId = resolved?.id ?? "";
  const notFound = !isPending && !error && !resolved;
  const { data: project } = useFetchProject(projectId);
  const section = isProjectSettingsSection(rawSection) ? rawSection : DEFAULT_PROJECT_SETTINGS_SECTION;
  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(resolved?.id);
  const crumbs: Crumb[] = projectCrumb ? [workspaceCrumb, projectCrumb] : [workspaceCrumb];

  return (
    <Container size="page" className="py-8">
      <PageHeader
        className="mb-8"
        crumbs={crumbs}
        title="Settings"
        meta={project && <span className="font-mono">{project.prefix}</span>}
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {notFound && (
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
      {project && <ProjectSettingsContent project={project} section={section} />}
    </Container>
  );
};
