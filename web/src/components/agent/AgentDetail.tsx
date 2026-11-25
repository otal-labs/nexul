import { AgentComputerDetail } from "@/components/agent/AgentComputerDetail";
import { AgentDefaultsPane } from "@/components/agent/AgentDefaultsPane";
import { AgentPlayDetail } from "@/components/agent/AgentPlayDetail";
import { AgentProjectPane } from "@/components/agent/AgentProjectPane";
import { DEFAULTS_ID, type AgentKind } from "@/models/Agent";

interface AgentDetailProps {
  kind: AgentKind;
  id: string;
}

export const AgentDetail = ({ kind, id }: AgentDetailProps) => (
  <div>
    {kind === "plays" && <AgentPlayDetail playId={id} />}
    {kind === "computers" && <AgentComputerDetail computerId={id} />}
    {kind === "projects" && id === DEFAULTS_ID && <AgentDefaultsPane />}
    {kind === "projects" && id !== DEFAULTS_ID && <AgentProjectPane projectId={id} />}
  </div>
);
