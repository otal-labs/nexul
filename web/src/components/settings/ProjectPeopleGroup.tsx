import { Microheader } from "@/components/Microheader";
import { ProjectPeopleRow } from "@/components/settings/ProjectPeopleRow";
import type { PermissionGrant } from "@/models/Permission";
import { grantKey } from "@/models/PermissionPanel";
import type { Project } from "@/models/Project";

interface ProjectPeopleGroupProps {
  title: string;
  project: Project;
  grants: PermissionGrant[];
  keptOut?: boolean;
}

export const ProjectPeopleGroup = ({ title, project, grants, keptOut }: ProjectPeopleGroupProps) => (
  <section className="space-y-1" aria-label={title}>
    <Microheader>{title}</Microheader>
    <ul>
      {grants.map((grant) => (
        <ProjectPeopleRow key={grantKey(grant)} project={project} grant={grant} keptOut={keptOut} />
      ))}
    </ul>
  </section>
);
