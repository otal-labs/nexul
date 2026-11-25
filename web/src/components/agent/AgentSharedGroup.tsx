import { useParams } from "react-router";

import { AgentSharedRow } from "@/components/agent/AgentSharedRow";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { sharedComputerName } from "@/models/SharedComputer";

interface AgentSharedGroupProps {
  query: string;
}

// Other people's computers shared with you by name; absent while there are none.
export const AgentSharedGroup = ({ query }: AgentSharedGroupProps) => {
  const { kind, id } = useParams();
  const { data: shared } = useListSharedComputers();
  const shown = shared?.filter((c) => query === "" || sharedComputerName(c).toLowerCase().includes(query));
  if (!shown || shown.length === 0) return null;

  return (
    <ListPaneGroup label="Shared with you" count={shown.length}>
      {shown.map((computer) => (
        <AgentSharedRow key={computer.id} computer={computer} selected={kind === "computers" && computer.id === id} />
      ))}
    </ListPaneGroup>
  );
};
