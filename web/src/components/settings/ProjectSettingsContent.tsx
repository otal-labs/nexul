import { ProjectCategories } from "@/components/project/ProjectCategories";
import { ProjectRepos } from "@/components/project/ProjectRepos";
import { ProjectServices } from "@/components/project/ProjectServices";
import { BoardSettingsSection } from "@/components/settings/BoardSettingsSection";
import { ProjectDangerZoneSection } from "@/components/settings/ProjectDangerZoneSection";
import { ProjectGeneralSection } from "@/components/settings/ProjectGeneralSection";
import { ProjectLinkSection } from "@/components/settings/ProjectLinkSection";
import { ProjectSettingsNav, type ProjectSettingsSection } from "@/components/settings/ProjectSettingsNav";
import type { Project } from "@/models/Project";

interface ProjectSettingsContentProps {
  project: Project;
  section: ProjectSettingsSection;
}

export const ProjectSettingsContent = ({ project, section }: ProjectSettingsContentProps) => (
  <div>
    <div className="mb-6 flex items-center gap-2 border-b border-border pb-6">
      <span className="font-mono text-sm text-muted-foreground">{project.prefix}</span>
      <span className="text-muted-foreground">/</span>
      <h1 className="text-lg font-semibold tracking-tight">{project.name}</h1>
    </div>
    <div className="flex flex-col gap-6 md:flex-row md:items-start md:gap-8">
      <ProjectSettingsNav active={section} />
      <div className="min-w-0 flex-1">
        {section === "general" && <ProjectGeneralSection project={project} />}
        {section === "categories" && <ProjectCategories projectId={project.id} />}
        {section === "repositories" && <ProjectRepos projectId={project.id} />}
        {section === "services" && <ProjectServices projectId={project.id} />}
        {section === "board" && <BoardSettingsSection projectId={project.id} />}
        {section === "pairing" && <ProjectLinkSection projectId={project.id} />}
        {section === "danger" && <ProjectDangerZoneSection project={project} />}
      </div>
    </div>
  </div>
);
