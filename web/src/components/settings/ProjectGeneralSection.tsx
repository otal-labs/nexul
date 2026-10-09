import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectName } from "@/components/project/ProjectName";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useRenameProject } from "@/hooks/ProjectHooks";
import type { Project } from "@/models/Project";

interface ProjectGeneralSectionProps {
  project: Project;
}

// The project's face as the sidebar and board show it; the prefix is fixed once set (ADR 0004), so only the name edits.
export const ProjectGeneralSection = ({ project }: ProjectGeneralSectionProps) => {
  const renameProject = useRenameProject();
  const created = new Date(project.created_at).toLocaleDateString(undefined, { dateStyle: "medium" });

  return (
    <SettingsCard id="general" title="General">
      <div className="flex items-center gap-4">
        <ProjectMark project={project} className="size-14 rounded-xl text-base" />
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex min-w-0 items-center gap-2 text-base">
            <ProjectName project={project} onRename={(name) => renameProject.mutateAsync({ id: project.id, name })} />
          </div>
          <p className="text-xs text-muted-foreground">
            Prefix <span className="font-mono text-foreground/90">{project.prefix}</span>, so tickets read{" "}
            <span className="font-mono">{project.prefix}-1</span>. It can't change.
          </p>
          <p className="font-mono text-xs text-muted-foreground">created {created}</p>
        </div>
      </div>
    </SettingsCard>
  );
};
