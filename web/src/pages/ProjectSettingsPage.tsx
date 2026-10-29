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
  sectionLabels,
} from "@/components/settings/ProjectSettingsNav";
import { ProjectSettingsContent } from "@/components/settings/ProjectSettingsContent";
import { ProjectSettingsMeta } from "@/components/settings/ProjectSettingsMeta";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";
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
  const canManageAccess = useCanManageAccess(resolved && { type: "project", id: resolved.id });
  // An unknown section, and Access for someone who can't change who gets in, land on General.
  const section =
    isProjectSettingsSection(rawSection) && (rawSection !== "access" || canManageAccess) ? rawSection : DEFAULT_PROJECT_SETTINGS_SECTION;
  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(resolved?.id);
  const crumbs: Crumb[] = projectCrumb ? [workspaceCrumb, projectCrumb, { label: "Settings" }] : [workspaceCrumb];

  return (
    <Container size="page" className="py-8">
      <PageHeader
        className="mb-8"
        crumbs={crumbs}
        // The open section names the page, so the title says where you are instead of repeating "Settings".
        title={sectionLabels[section]}
        meta={project && <ProjectSettingsMeta project={project} section={section} />}
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
      {project && <ProjectSettingsContent project={project} section={section} canManageAccess={canManageAccess} />}
    </Container>
  );
};
