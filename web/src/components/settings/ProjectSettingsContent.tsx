import { ProjectCategories } from "@/components/project/ProjectCategories";
import { ProjectRepos } from "@/components/project/ProjectRepos";
import { ProjectServices } from "@/components/project/ProjectServices";
import { BoardSettingsSection } from "@/components/settings/BoardSettingsSection";
import { ProjectDangerZoneSection } from "@/components/settings/ProjectDangerZoneSection";
import { ProjectGeneralSection } from "@/components/settings/ProjectGeneralSection";
import { ProjectPeopleAccessSection } from "@/components/settings/ProjectPeopleAccessSection";
import { ProjectSettingsNav, type ProjectSettingsSection } from "@/components/settings/ProjectSettingsNav";
import { SettingsShell } from "@/components/settings/SettingsShell";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";

interface ProjectSettingsContentProps {
  project: Project;
  section: ProjectSettingsSection;
}

export const ProjectSettingsContent = ({ project, section }: ProjectSettingsContentProps) => {
  const canManageMembers = useHasPermission("members:write");
  return (
    <SettingsShell section={section} nav={<ProjectSettingsNav project={project} active={section} />}>
      {section === "general" && <ProjectGeneralSection project={project} />}
      {section === "general" && canManageMembers && <ProjectPeopleAccessSection project={project} />}
      {section === "categories" && <ProjectCategories projectId={project.id} />}
      {section === "repositories" && <ProjectRepos projectId={project.id} />}
      {section === "services" && <ProjectServices projectId={project.id} />}
      {section === "board" && <BoardSettingsSection projectId={project.id} />}
      {section === "danger" && <ProjectDangerZoneSection project={project} />}
    </SettingsShell>
  );
};
