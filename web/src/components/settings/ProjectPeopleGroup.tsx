import { Microheader } from "@/components/Microheader";
import { ProjectPeopleRow } from "@/components/settings/ProjectPeopleRow";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
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
  <section aria-label={title}>
    <Microheader className="flex h-8 items-center">{title}</Microheader>
    <ul className={settingsRowListClass}>
      {grants.map((grant) => (
        <ProjectPeopleRow key={grantKey(grant)} project={project} grant={grant} keptOut={keptOut} />
      ))}
    </ul>
  </section>
);
