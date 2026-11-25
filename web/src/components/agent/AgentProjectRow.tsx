import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { ProjectMark } from "@/components/project/ProjectMark";
import { useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPath } from "@/models/Agent";
import type { Project } from "@/models/Project";

interface AgentProjectRowProps {
  project: Project;
  selected: boolean;
}

// A project and what its runs use: "Uses defaults" until its link changes anything.
export const AgentProjectRow = ({ project, selected }: AgentProjectRowProps) => {
  const wsPath = useWorkspacePath();
  const { data: links } = useFetchProjectLinks();
  const summary = useProjectLinkSummary(links?.find((l) => l.project_id === project.id));

  return (
    <ListPaneRow
      to={wsPath(agentPath("projects", project.id))}
      title={project.name}
      leading={<ProjectMark project={project} className="size-6 rounded-md text-micro shadow-none" />}
      snippet={summary ?? ""}
      selected={selected}
    />
  );
};
