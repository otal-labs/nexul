import { Suspense, useState } from "react";
import { SquareTerminalIcon } from "lucide-react";

import { LazyTerminalDialog } from "@/components/terminal/LazyTerminal";
import { Button } from "@/components/ui/button";
import { useOpenComputerTerminal } from "@/hooks/ComputerCommandHooks";
import { terminalSocketPath, type ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";

interface ComputerTerminalButtonProps {
  computer: Computer;
  // Offline: the button keeps its place, so a header has one shape.
  offline?: boolean;
}

// Opens a shell on the computer through its Nexul app, in its own dialog.
export const ComputerTerminalButton = ({ computer, offline = false }: ComputerTerminalButtonProps) => {
  const openTerminal = useOpenComputerTerminal(computer.id);
  const [terminal, setTerminal] = useState<ComputerCommand | null>(null);

  return (
    <>
      <span {...(offline && { title: "Computer is offline" })}>
        <Button type="button" variant="outline" size="sm" disabled={offline} loading={openTerminal.isPending} onClick={() => openTerminal.mutate(undefined, { onSuccess: setTerminal })}>
          <SquareTerminalIcon aria-hidden /> Open terminal
        </Button>
      </span>
      {terminal && (
        <Suspense fallback={null}>
          <LazyTerminalDialog title={computer.name} socketPath={terminalSocketPath(terminal)} onClose={() => setTerminal(null)} />
        </Suspense>
      )}
    </>
  );
};
