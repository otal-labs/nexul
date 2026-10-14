import { ActionButton } from "@/components/action/ActionButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { inSetup, type Project } from "@/models/Project";
import { setupResumePath } from "@/models/ProjectWizard";

interface ProjectSetupSectionProps {
  project: Project;
}

// The way back into the wizard once the sidebar no longer offers it; revisiting a step leaves the project set up.
export const ProjectSetupSection = ({ project }: ProjectSetupSectionProps) => {
  const wsPath = useWorkspacePath();
  const description = inSetup(project)
    ? "Not finished. The sidebar offers Continue setup until someone presses Finish."
    : "Finished. Revisit any step of the wizard; the project stays set up.";

  return (
    <SettingsCard id="setup" title="Setup" description={description}>
      <ActionButton permission="projects:write" projectId={project.id} verb="change a project's setup" size="sm" variant="outline" to={wsPath(`${setupResumePath(project)}&revisit=1`)}>
        Open the wizard
      </ActionButton>
    </SettingsCard>
  );
};
