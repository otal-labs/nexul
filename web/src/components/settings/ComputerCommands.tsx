import { useState, type FormEvent } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ComputerCommandRow } from "@/components/settings/ComputerCommandRow";
import { ComputerCommandSheet } from "@/components/settings/ComputerCommandSheet";
import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { useFetchComputerCommands, useRunComputerCommand } from "@/hooks/ComputerCommandHooks";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import type { ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";

interface ComputerCommandsProps {
  computer: Computer;
}

// Commands run on the computer through its Nexul app, from here or by an agent's command_run. Its owner has the switch
// and every command; a person it is shared with by Run commands sees only their own.
export const ComputerCommands = ({ computer }: ComputerCommandsProps) => {
  const { data: commands, isPending, error } = useFetchComputerCommands(computer.id);
  const settings = useUpdateComputerSettings(computer.id);
  const run = useRunComputerCommand(computer.id);
  const [shown, setShown] = useState<{ command: ComputerCommand | null; open: boolean }>({ command: null, open: false });
  const owner = !computer.shared;
  const offThere = computer.facts?.shell?.enabled === false;

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = event.currentTarget;
    const command = String(new FormData(form).get("command") ?? "").trim();
    if (!command) return;
    run.mutate(command, { onSuccess: () => form.reset() });
  };

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
        {offThere && (
          <SettingsStatus tone="muted" detail={<span className="font-mono">nexul shell on</span>}>
            Off on the computer
          </SettingsStatus>
        )}
        {(computer.shell_enabled || !owner) && (
          <form className="flex items-center gap-2" onSubmit={submit}>
            <Input name="command" aria-label="Command" placeholder="Command" autoComplete="off" spellCheck={false} className="font-mono" />
            <Button type="submit" variant="outline" size="sm" disabled={run.isPending}>
              Run
            </Button>
          </form>
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
    </ComputerFact>
  );
};
