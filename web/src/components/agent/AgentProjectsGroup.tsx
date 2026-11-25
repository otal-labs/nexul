import { useParams } from "react-router";

import { AgentDefaultsRow } from "@/components/agent/AgentDefaultsRow";
import { AgentProjectRow } from "@/components/agent/AgentProjectRow";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { DEFAULTS_ID } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AgentProjectsGroupProps {
  query: string;
}

// Where each of this workspace's projects runs: Defaults first, which every project uses until it changes something.
export const AgentProjectsGroup = ({ query }: AgentProjectsGroupProps) => {
  const { kind, id } = useParams();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects } = useFetchWorkspaceProjects(workspaceId);
  const shown = projects?.filter((p) => query === "" || p.name.toLowerCase().includes(query)) ?? [];
  const defaults = query === "" || "defaults".includes(query);
  const selectedId = kind === "projects" ? id : undefined;
  if (!defaults && shown.length === 0) return null;

  return (
    <ListPaneGroup label="Projects" count={shown.length}>
      {defaults && <AgentDefaultsRow selected={selectedId === DEFAULTS_ID} />}
      {shown.map((project) => (
        <AgentProjectRow key={project.id} project={project} selected={project.id === selectedId} />
      ))}
    </ListPaneGroup>
  );
};
