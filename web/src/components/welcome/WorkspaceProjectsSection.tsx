import { projectPermissions } from "@nexul/client-core/permissions";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { microheaderClass } from "@/components/Microheader";
import { ProjectAccessRow } from "@/components/welcome/ProjectAccessRow";
import { WizardCardList } from "@/components/wizard/WizardCardList";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { domainsIn } from "@/models/PermissionLevel";
import type { Workspace } from "@/models/Workspace";
import { cn } from "@/lib/utils";

interface WorkspaceProjectsSectionProps {
  workspace: Workspace;
  // Named only when the person belongs to more than one workspace.
  named: boolean;
}

export const WorkspaceProjectsSection = ({ workspace, named }: WorkspaceProjectsSectionProps) => {
  const { data: projects, error } = useFetchWorkspaceProjects(workspace.id);
  const { data: mine } = useFetchMyRole(workspace.id);
  const { data: catalog } = useFetchPermissionCatalog();
  const domains = domainsIn(catalog ?? [], ["project"]).filter((domain) => domain.levels.length > 0);
  // Levels wait for the role, or every row reads None first; the section holds its place meanwhile so others don't jump.
  const ready = !!projects && !!mine && !!catalog;
  // A workspace the person is in no project of has nothing to list here.
  if (projects && projects.length === 0) return null;

  return (
    <section>
      {named && <h2 className={cn(microheaderClass, "mb-2")}>{workspace.name}</h2>}
      {error && <ErrorDisplay error={error} />}
      {!error && !ready && <LoadingDisplay />}
      {ready && (
        <WizardCardList>
          {projects.map((project) => (
            <ProjectAccessRow
              key={project.id}
              project={project}
              workspaceSlug={workspace.slug}
              domains={domains}
              held={projectPermissions(mine, project.id)}
            />
          ))}
        </WizardCardList>
      )}
    </section>
  );
};
