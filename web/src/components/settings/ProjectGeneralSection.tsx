import { ActionButton } from "@/components/action/ActionButton";
import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectName } from "@/components/project/ProjectName";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useRenameProject } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { inSetup, type Project } from "@/models/Project";
import { setupResumePath } from "@/models/ProjectWizard";

interface ProjectGeneralSectionProps {
  project: Project;
}

// The project's face as the sidebar and board show it, then when it was made and its setup; the prefix is fixed once set (ADR 0004).
export const ProjectGeneralSection = ({ project }: ProjectGeneralSectionProps) => {
  const renameProject = useRenameProject();
  const wsPath = useWorkspacePath();
  const created = new Date(project.created_at).toLocaleDateString(undefined, { dateStyle: "medium" });
  const unfinished = inSetup(project);

  return (
    <SettingsCard id="project" title="Project">
      <div className="flex items-center gap-4 pb-5">
        <ProjectMark project={project} className="size-12 rounded-lg text-sm" />
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex min-w-0 items-center gap-2 text-sm font-medium">
            <ProjectName project={project} onRename={(name) => renameProject.mutateAsync({ id: project.id, name })} />
          </div>
          <SettingsHint>
            Prefix <span className="font-mono text-foreground/90">{project.prefix}</span>, so tickets read{" "}
            <span className="font-mono">{project.prefix}-1</span>. It can't change.
          </SettingsHint>
        </div>
      </div>
      <SettingsRows className="-mx-2 [&>*]:px-2">
        <SettingsRow label="Created">
          <span className="font-mono text-xs text-muted-foreground tabular-nums">{created}</span>
        </SettingsRow>
        <SettingsRow
          label="Setup"
          description={unfinished ? "The sidebar shows Continue setup until it is finished." : "Revisit any step; the project stays set up."}
        >
          <div className="flex items-center gap-3">
            <SettingsStatus tone={unfinished ? "warning" : "success"}>{unfinished ? "Not finished" : "Finished"}</SettingsStatus>
            <ActionButton permission="projects:write" projectId={project.id} verb="change a project's setup" size="sm" variant="outline" to={wsPath(`${setupResumePath(project)}&revisit=1`)}>
              Open the wizard
            </ActionButton>
          </div>
        </SettingsRow>
      </SettingsRows>
    </SettingsCard>
  );
};
