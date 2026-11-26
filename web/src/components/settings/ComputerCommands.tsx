import { Suspense, useState } from "react";

import { SquareTerminalIcon } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ComputerCommandRow } from "@/components/settings/ComputerCommandRow";
import { ComputerCommandSheet } from "@/components/settings/ComputerCommandSheet";
import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { LazyTerminalDialog } from "@/components/terminal/LazyTerminal";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useFetchComputerCommands, useOpenComputerTerminal } from "@/hooks/ComputerCommandHooks";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { terminalSocketPath, type ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";

interface ComputerCommandsProps {
  computer: Computer;
}

// A terminal on the computer through its Nexul app, and the record of it and of an agent's command_run. Its owner has the
// switch and every entry; a person it is shared with by Run commands sees only their own.
export const ComputerCommands = ({ computer }: ComputerCommandsProps) => {
  const { data: commands, isPending, error } = useFetchComputerCommands(computer.id);
  const settings = useUpdateComputerSettings(computer.id);
  const openTerminal = useOpenComputerTerminal(computer.id);
  const [terminal, setTerminal] = useState<ComputerCommand | null>(null);
  const [shown, setShown] = useState<{ command: ComputerCommand | null; open: boolean }>({ command: null, open: false });
  const owner = !computer.shared;
  const offThere = computer.facts?.shell?.enabled === false;

  return (
    <ComputerFact label="Commands">
      <div className="space-y-2">
        {owner && (
          <label className="flex items-center gap-2 text-xs">
            <Switch
              size="sm"
              checked={computer.shell_enabled}
              disabled={settings.isPending}
              onCheckedChange={(on) => settings.mutate({ shell_enabled: on })}
            />
            <span className="text-foreground">Run commands from Nexul</span>
          </label>
        )}
        {owner && (
          <p className="text-xs text-muted-foreground">Commands run here are recorded and can be read by people with Read computer activity.</p>
        )}
        {offThere && (
          <SettingsStatus tone="muted" detail={<span className="font-mono">nexul shell on</span>}>
            Off on the computer
          </SettingsStatus>
        )}
        {(computer.shell_enabled || !owner) && (
          <Button type="button" variant="outline" size="sm" loading={openTerminal.isPending} onClick={() => openTerminal.mutate(undefined, { onSuccess: setTerminal })}>
            <SquareTerminalIcon aria-hidden /> Open terminal
          </Button>
        )}
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
        <ComputerCommandSheet command={shown.command} open={shown.open} onOpenChange={(open) => setShown((prev) => ({ ...prev, open }))} />
      )}
      {terminal && (
        <Suspense fallback={null}>
          <LazyTerminalDialog title={computer.name} socketPath={terminalSocketPath(terminal)} onClose={() => setTerminal(null)} />
        </Suspense>
      )}
    </ComputerFact>
  );
};
