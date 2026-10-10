import { useState } from "react";

import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { pairPlan, T3_INSTALL_LABELS, T3Install } from "@/utils/PairCommands";
import { PAIRING_GUIDE } from "@/utils/TunnelInstallCommands";

interface PairCommandsProps {
  // A computer with a tunnel ran the tunnel command, which installs T3 Code where it was missing.
  viaTunnel: boolean;
}

// The command that prints a pairing token depends on how T3 Code got onto the computer: the desktop app's `t3` is off PATH.
export const PairCommands = ({ viaTunnel }: PairCommandsProps) => {
  const [install, setInstall] = useState<T3Install>(T3Install.Desktop);
  const plan = pairPlan(install, viaTunnel);
  return (
    <div className="space-y-3">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label="How T3 Code is installed"
        value={install}
        onValueChange={(next) => next && setInstall(next as T3Install)}
      >
        {Object.values(T3Install).map((value) => (
          <ToggleGroupItem key={value} value={value}>
            {T3_INSTALL_LABELS[value]}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <p className="text-sm text-muted-foreground">{plan.lead}</p>
      <CommandSnippet key={install} commands={plan.commands} label="Pairing command" guide={PAIRING_GUIDE} />
    </div>
  );
};
