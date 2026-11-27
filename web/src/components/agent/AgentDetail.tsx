import { AgentComputerDetail } from "@/components/agent/AgentComputerDetail";
import { AgentDefaultsPane } from "@/components/agent/AgentDefaultsPane";
import { AgentProjectPane } from "@/components/agent/AgentProjectPane";
import { DEFAULTS_ID } from "@/models/Agent";

interface AgentDetailProps {
  kind: "computers" | "projects";
  id: string;
}

export const AgentDetail = ({ kind, id }: AgentDetailProps) => (
  <div>
    {kind === "computers" && <AgentComputerDetail computerId={id} />}
    {kind === "projects" && id === DEFAULTS_ID && <AgentDefaultsPane />}
    {kind === "projects" && id !== DEFAULTS_ID && <AgentProjectPane projectId={id} />}
  </div>
);
