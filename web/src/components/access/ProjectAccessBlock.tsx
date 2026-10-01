import { EveryProjectRow } from "@/components/access/EveryProjectRow";
import { ProjectAccessList, type AccessProject } from "@/components/access/ProjectAccessList";
import { EveryProject } from "@/models/Team";

interface ProjectAccessBlockProps {
  subject: string;
  everyProject: EveryProject;
  onEveryProject: (value: EveryProject) => void;
  projects: AccessProject[];
  access: Record<string, string[]>;
  onProjectAccess: (projectId: string, allow: string[]) => void;
  disabled?: boolean;
}

// The Every project row, then, for someone held to chosen projects, one row per project: the Team dialog and
// invitations share it.
export const ProjectAccessBlock = ({ subject, everyProject, onEveryProject, projects, access, onProjectAccess, disabled = false }: ProjectAccessBlockProps) => (
  <div>
    <EveryProjectRow subject={subject} value={everyProject} onChange={onEveryProject} disabled={disabled} />
    {everyProject === EveryProject.None && (
      <div className="border-t border-border">
        <ProjectAccessList projects={projects} access={access} onChange={onProjectAccess} disabled={disabled} />
      </div>
    )}
  </div>
);
