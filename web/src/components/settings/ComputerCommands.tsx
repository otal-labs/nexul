import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ComputerCommandRow } from "@/components/settings/ComputerCommandRow";
import { ComputerCommandDialog } from "@/components/settings/ComputerCommandDialog";
import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { ComputerTerminalButton } from "@/components/computers/ComputerTerminalButton";
import { useFetchComputerCommands } from "@/hooks/ComputerCommandHooks";
import type { ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";

interface ComputerCommandsProps {
  computer: Computer;
}

// The record of terminals on the computer and of an agent's command_run. Its owner (whose switch and Open terminal sit in
// the computer's header) sees every entry; a person it is shared with by Run commands opens a terminal here and sees only their own.
export const ComputerCommands = ({ computer }: ComputerCommandsProps) => {
  const { data: commands, isPending, error } = useFetchComputerCommands(computer.id);
  const [shown, setShown] = useState<{ command: ComputerCommand | null; open: boolean }>({ command: null, open: false });
  const owner = !computer.shared;
  const offThere = computer.facts?.shell?.enabled === false;

  return (
    <ComputerFact label="Commands">
      <div className="space-y-2">
        {owner && (
          <p className="text-xs text-muted-foreground">Commands run here are recorded and can be read by people with Read computer activity.</p>
        )}
        {offThere && (
          <SettingsStatus tone="muted" detail={<span className="font-mono">nexul shell on</span>}>
            Off on the computer
          </SettingsStatus>
        )}
        {!owner && <ComputerTerminalButton computer={computer} />}
        {isPending && <LoadingDisplay label="Loading commands" className="justify-start p-0" />}
        {error && <ErrorDisplay error={error} title="Couldn't list the commands." className="p-3" />}
        {commands && commands.length === 0 && <EmptyRow flush className="text-xs">No commands have run here.</EmptyRow>}
        {commands && commands.length > 0 && (
          <ul className="divide-y divide-border">
            {commands.map((command) => (
              <ComputerCommandRow key={command.id} command={command} named={owner} onOutput={(c) => setShown({ command: c, open: true })} />
            ))}
          </ul>
        )}
      </div>
      {shown.command && (
        <ComputerCommandDialog command={shown.command} open={shown.open} onOpenChange={(open) => setShown((prev) => ({ ...prev, open }))} />
      )}
    </ComputerFact>
  );
};
