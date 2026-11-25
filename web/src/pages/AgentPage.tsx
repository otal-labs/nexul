import { SparklesIcon } from "lucide-react";
import { useParams } from "react-router";

import { AgentDetail } from "@/components/agent/AgentDetail";
import { AgentListPane } from "@/components/agent/AgentListPane";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { isAgentKind } from "@/models/Agent";

// Plays, your computers, the ones shared with you, and where each project's runs go, beside the open one (ADR 0211).
export const AgentPage = () => {
  const { kind, id } = useParams();
  const open = isAgentKind(kind) && !!id;

  return (
    <div className="h-full">
      <ListDetailLayout
        hasSelection={open}
        stableGutter
        list={<AgentListPane />}
        placeholder={<ListDetailPlaceholder icon={SparklesIcon} title="Select a play or a computer" summary="What the Agent runs, and where" />}
        detail={open && <AgentDetail key={`${kind}/${id}`} kind={kind} id={id} />}
      />
    </div>
  );
};
