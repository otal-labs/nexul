import { useState } from "react";

import { CommandBlock } from "@/components/pairing/CommandBlock";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { pairPlan, T3_INSTALL_LABELS, T3Install } from "@/utils/PairCommands";
import { detectTunnelOs } from "@/utils/TunnelInstallCommands";

// The command that prints a pairing token depends on how T3 Code got onto the computer: the desktop app's `t3` is off PATH.
export const PairCommands = () => {
  const [install, setInstall] = useState<T3Install>(T3Install.Desktop);
  const plan = pairPlan(install, detectTunnelOs(navigator.userAgent));
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
      <CommandBlock key={install} shell={plan.shell} lines={plan.lines} />
      {plan.other && (
        <p className="text-xs text-muted-foreground">
          {plan.other.label} <code className="font-mono break-all text-foreground">{plan.other.line}</code>
        </p>
      )}
    </div>
  );
};
