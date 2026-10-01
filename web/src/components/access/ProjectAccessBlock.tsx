import { EveryProjectRow } from "@/components/access/EveryProjectRow";
import { ProjectAccessRow } from "@/components/access/ProjectAccessRow";
import { EmptyRow } from "@/components/EmptyRow";
import { PermissionLockContext } from "@/hooks/usePermissionLock";
import { EveryProject } from "@/models/Team";

export interface AccessProject {
  id: string;
  name: string;
}

interface ProjectAccessBlockProps {
  subject: string;
  everyProject: EveryProject;
  onEveryProject: (value: EveryProject) => void;
  projects: AccessProject[];
  access: Record<string, string[]>;
  onProjectAccess: (projectId: string, allow: string[]) => void;
  disabled?: boolean;
}

// The project list in the shape of a role's domain list: Every project leads, one row per project under it, held while
// it reads From role. The Team dialog and invitations share it.
export const ProjectAccessBlock = ({ subject, everyProject, onEveryProject, projects, access, onProjectAccess, disabled = false }: ProjectAccessBlockProps) => (
  <div className="space-y-2">
    <ul className="@container divide-y divide-border overflow-hidden rounded-md border border-input">
      <EveryProjectRow subject={subject} value={everyProject} onChange={onEveryProject} disabled={disabled} />
      <PermissionLockContext value={disabled || everyProject === EveryProject.Role}>
        {projects.map((project) => (
          <ProjectAccessRow key={project.id} name={project.name} value={access[project.id] ?? []} onChange={(allow) => onProjectAccess(project.id, allow)} />
        ))}
      </PermissionLockContext>
    </ul>
    {projects.length === 0 && <EmptyRow>No projects to give access to yet.</EmptyRow>}
  </div>
);
