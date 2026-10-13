import { projectPermissions } from "@nexul/client-core/permissions";

import { microheaderClass } from "@/components/Microheader";
import { ProjectAccessRow } from "@/components/welcome/ProjectAccessRow";
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
  const { data: projects } = useFetchWorkspaceProjects(workspace.id);
  const { data: mine } = useFetchMyRole(workspace.id);
  const { data: catalog } = useFetchPermissionCatalog();
  const domains = domainsIn(catalog ?? [], ["project"]).filter((domain) => domain.levels.length > 0);
  // A workspace the person is in no project of has nothing to list here.
  if (!projects || projects.length === 0) return null;

  return (
    <section>
      {named && <h2 className={cn(microheaderClass, "pt-4")}>{workspace.name}</h2>}
      <ul className="divide-y divide-border">
        {projects.map((project) => (
          <ProjectAccessRow
            key={project.id}
            project={project}
            workspaceSlug={workspace.slug}
            domains={domains}
            held={projectPermissions(mine, project.id)}
          />
        ))}
      </ul>
    </section>
  );
};
