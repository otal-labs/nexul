import { ProjectIconPicker } from "@/components/project/ProjectIconPicker";
import { ProjectName } from "@/components/project/ProjectName";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useRenameProject } from "@/hooks/ProjectHooks";
import type { Project } from "@/models/Project";

interface ProjectGeneralSectionProps {
  project: Project;
}

// Prefix is read-only — immutable once set (ADR 0004), so there's nothing to edit.
export const ProjectGeneralSection = ({ project }: ProjectGeneralSectionProps) => {
  const renameProject = useRenameProject();

  return (
    <SettingsCard id="general" title="General">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 sm:divide-x sm:divide-border">
        <div className="sm:pr-6">
          <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Icon & Name</p>
          <div className="mt-2 flex items-center gap-2">
            <ProjectIconPicker
              icon={project.icon ?? ""}
              prefix={project.prefix}
              onChange={(icon) => void renameProject.mutateAsync({ id: project.id, name: project.name, icon })}
            />
            <ProjectName
              project={project}
              className="opacity-100"
              onRename={(name) =>
                renameProject.mutateAsync({ id: project.id, name, icon: project.icon ?? "" })
              }
            />
          </div>
        </div>
        <div className="sm:pl-6">
          <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Identifier</p>
          <p className="mt-2 font-mono text-sm">{project.prefix}</p>
          <p className="mt-1 text-xs text-muted-foreground">Used in ticket IDs</p>
        </div>
      </div>
    </SettingsCard>
  );
};
