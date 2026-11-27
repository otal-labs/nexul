import { useParams } from "react-router";

import { AgentDetail } from "@/components/agent/AgentDetail";
import { AgentListPane } from "@/components/agent/AgentListPane";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";

// The computer and project panes the Computers dialog is replacing, in the list-and-detail page they lived in.
export const LegacyAgentPage = () => {
  const { kind, id } = useParams();
  const open = (kind === "computers" || kind === "projects") && !!id;
  return (
    <div className="h-full">
      <ListDetailLayout hasSelection={open} stableGutter list={<AgentListPane />} placeholder={null} detail={open && <AgentDetail key={`${kind}/${id}`} kind={kind} id={id} />} />
    </div>
  );
};
