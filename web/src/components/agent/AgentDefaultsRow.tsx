import { SlidersHorizontalIcon } from "lucide-react";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useDefaultsSummary } from "@/hooks/PairingProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPath, DEFAULTS_ID } from "@/models/Agent";

interface AgentDefaultsRowProps {
  selected: boolean;
}

// What @Agent and every project's runs use until a project changes it.
export const AgentDefaultsRow = ({ selected }: AgentDefaultsRowProps) => {
  const wsPath = useWorkspacePath();
  const { data: defaults, error } = useFetchPairingDefaults();
  const summary = useDefaultsSummary(defaults);

  return (
    <ListPaneRow
      to={wsPath(agentPath("projects", DEFAULTS_ID))}
      title="Defaults"
      leading={<SlidersHorizontalIcon aria-hidden className="size-3.5 text-muted-foreground" />}
      snippet={error ? "Couldn't load" : (summary ?? "")}
      selected={selected}
    />
  );
};
