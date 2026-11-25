import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { statusDotClass } from "@/components/settings/SettingsStatus";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { agentPath } from "@/models/Agent";
import { runnerPresence } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";
import { sharedComputerAbilities } from "@/models/SharedComputer";

interface AgentSharedRowProps {
  computer: Computer;
  selected: boolean;
}

// The group already says it is shared, so the row names the owner and what they let you do.
export const AgentSharedRow = ({ computer, selected }: AgentSharedRowProps) => {
  const wsPath = useWorkspacePath();
  const presence = runnerPresence(computer.runner);

  return (
    <ListPaneRow
      to={wsPath(agentPath("computers", computer.id))}
      title={computer.shared ? `${computer.shared.owner_name}'s ${computer.name}` : computer.name}
      leading={<span aria-label={presence.text} role="img" className={cn("size-1.5 rounded-full", statusDotClass[presence.tone])} />}
      snippet={computer.shared ? sharedComputerAbilities(computer.shared) : ""}
      selected={selected}
    />
  );
};
