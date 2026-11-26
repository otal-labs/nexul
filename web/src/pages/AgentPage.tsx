import { SparklesIcon } from "lucide-react";
import { Navigate, useParams } from "react-router";

import { AgentDetail } from "@/components/agent/AgentDetail";
import { AgentListPane } from "@/components/agent/AgentListPane";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { useFirstPlayPath } from "@/hooks/useFirstPlayPath";
import { isAgentKind } from "@/models/Agent";

// Plays, your computers, the ones shared with you, and where each project's runs go, beside the open one (ADR 0211).
export const AgentPage = () => {
  const { kind, id } = useParams();
  const open = isAgentKind(kind) && !!id;
  const first = useFirstPlayPath();

  return (
    <div className="h-full">
      {!open && first.to && <Navigate replace to={first.to} />}
      <ListDetailLayout
        hasSelection={open}
        stableGutter
        list={<AgentListPane />}
        placeholder={!first.pending && !first.to && <ListDetailPlaceholder icon={SparklesIcon} title="Select a play or a computer" summary="What the Agent runs, and where" />}
        detail={open && <AgentDetail key={`${kind}/${id}`} kind={kind} id={id} />}
      />
    </div>
  );
};
