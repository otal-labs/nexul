import { Suspense, useState } from "react";
import { SquareTerminalIcon } from "lucide-react";

import { LazyTerminalDialog } from "@/components/terminal/LazyTerminal";
import { Button } from "@/components/ui/button";
import { useOpenComputerTerminal } from "@/hooks/ComputerCommandHooks";
import { terminalSocketPath, type ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";

interface ComputerTerminalButtonProps {
  computer: Computer;
}

// Opens a shell on the computer through its Nexul app, in its own dialog.
export const ComputerTerminalButton = ({ computer }: ComputerTerminalButtonProps) => {
  const openTerminal = useOpenComputerTerminal(computer.id);
  const [terminal, setTerminal] = useState<ComputerCommand | null>(null);

  return (
    <>
      <Button type="button" variant="outline" size="sm" loading={openTerminal.isPending} onClick={() => openTerminal.mutate(undefined, { onSuccess: setTerminal })}>
        <SquareTerminalIcon aria-hidden /> Open terminal
      </Button>
      {terminal && (
        <Suspense fallback={null}>
          <LazyTerminalDialog title={computer.name} socketPath={terminalSocketPath(terminal)} onClose={() => setTerminal(null)} />
        </Suspense>
      )}
    </>
  );
};
