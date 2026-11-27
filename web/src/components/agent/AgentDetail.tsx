import { AgentPlayDetail } from "@/components/agent/AgentPlayDetail";
import type { AgentKind } from "@/models/Agent";

interface AgentDetailProps {
  kind: AgentKind;
  id: string;
}

export const AgentDetail = ({ kind, id }: AgentDetailProps) => <div>{kind === "plays" && <AgentPlayDetail playId={id} />}</div>;
