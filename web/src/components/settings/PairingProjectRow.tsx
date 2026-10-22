import { ProjectMark } from "@/components/project/ProjectMark";
import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import { RunSettingsRow } from "@/components/settings/RunSettingsRow";
import { useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import type { Project } from "@/models/Project";

interface PairingProjectRowProps {
  project: Project;
  open: boolean;
  onToggle: () => void;
}

// A project in the list: what its runs use, "Uses defaults" until its link changes anything, and its editor.
export const PairingProjectRow = ({ project, open, onToggle }: PairingProjectRowProps) => {
  const { data: links } = useFetchProjectLinks();
  const link = links?.find((l) => l.project_id === project.id);
  const summary = useProjectLinkSummary(link);

  return (
    <RunSettingsRow
      mark={<ProjectMark project={project} className="size-7 rounded-md text-[10px] shadow-none" />}
      title={project.name}
      summary={summary}
      open={open}
      onToggle={onToggle}
    >
      <ProjectLinkForm projectId={project.id} projectName={project.name} link={link ?? { project_id: project.id, computers: [], checkouts: [] }} />
    </RunSettingsRow>
  );
};
