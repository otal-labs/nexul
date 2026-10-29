import { ProjectCategories } from "@/components/project/ProjectCategories";
import { ProjectRepos } from "@/components/project/ProjectRepos";
import { ProjectServices } from "@/components/project/ProjectServices";
import { BoardLabelColorsSection } from "@/components/settings/BoardLabelColorsSection";
import { BoardStatusColumnsSection } from "@/components/settings/BoardStatusColumnsSection";
import { BoardTicketTypesSection } from "@/components/settings/BoardTicketTypesSection";
import { ProjectDangerZoneSection } from "@/components/settings/ProjectDangerZoneSection";
import { ProjectGeneralSection } from "@/components/settings/ProjectGeneralSection";
import { ProjectPeopleSection } from "@/components/settings/ProjectPeopleSection";
import { ProjectSettingsNav, type ProjectSettingsSection } from "@/components/settings/ProjectSettingsNav";
import { SettingsShell } from "@/components/settings/SettingsShell";
import type { Project } from "@/models/Project";

interface ProjectSettingsContentProps {
  project: Project;
  section: ProjectSettingsSection;
  canManageAccess: boolean;
}

export const ProjectSettingsContent = ({ project, section, canManageAccess }: ProjectSettingsContentProps) => (
  <SettingsShell section={section} nav={<ProjectSettingsNav project={project} active={section} canManageAccess={canManageAccess} />}>
    {section === "general" && <ProjectGeneralSection project={project} />}
    {section === "access" && <ProjectPeopleSection project={project} />}
    {section === "statuses" && <BoardStatusColumnsSection projectId={project.id} />}
    {section === "categories" && <ProjectCategories projectId={project.id} />}
    {section === "types" && <BoardTicketTypesSection projectId={project.id} />}
    {section === "labels" && <BoardLabelColorsSection projectId={project.id} />}
    {section === "repositories" && <ProjectRepos projectId={project.id} />}
    {section === "services" && <ProjectServices projectId={project.id} />}
    {section === "danger" && <ProjectDangerZoneSection project={project} />}
  </SettingsShell>
);
