import { Suspense, useState } from "react";

import { SquareTerminalIcon } from "lucide-react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { ActionSwitch } from "@/components/action/ActionSwitch";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ComputerCommandRow } from "@/components/settings/ComputerCommandRow";
import { ComputerCommandSheet } from "@/components/settings/ComputerCommandSheet";
import { LazyTerminalDialog } from "@/components/terminal/LazyTerminal";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchMachineCommands, useOpenMachineTerminal, useSetMachineShell } from "@/hooks/MachineCommandHooks";
import { useRunners } from "@/hooks/RunnerHooks";
import { terminalSocketPath, type ComputerCommand } from "@/models/ComputerCommand";
import type { Machine } from "@/models/Runner";

interface MachineCommandsProps {
  machine: Machine;
}

// Commands from Nexul on a machine's runners, for holders of Run commands there: the machine's switch, a terminal on
// one runner, and the newest terminals and commands with who ran them and where.
export const MachineCommands = ({ machine }: MachineCommandsProps) => {
  const on = !!machine.shell_enabled;
  const { data: commands, isPending, error } = useFetchMachineCommands(machine.id, true);
  const { data: allRunners } = useRunners();
  const setShell = useSetMachineShell(machine.id);
  const openTerminal = useOpenMachineTerminal(machine.id);
  const [terminal, setTerminal] = useState<ComputerCommand | null>(null);
  const runners = (allRunners ?? []).filter((r) => r.machine === machine.name);
  const [picked, setPicked] = useState("");
  const runnerId = picked || runners.find((r) => r.connected)?.id || runners[0]?.id || "";
  const [shown, setShown] = useState<{
    command: ComputerCommand | null;
    open: boolean;
  }>({ command: null, open: false });
  const gate = {
    permission: "machines:shell",
    answer: machineAnswer(machine, "machines:shell"),
    verb: "run commands on this machine",
  } as const;

  const runnerName = runners.find((r) => r.id === runnerId)?.name ?? machine.name;

  return (
    <div className="space-y-3 text-sm">
      <label className="flex items-center gap-2 text-xs">
        <ActionSwitch
          {...gate}
          size="sm"
          checked={on}
          disabled={setShell.isPending}
          onCheckedChange={(next) => setShell.mutate(next)}
          aria-label="Run commands from Nexul"
        />
        <span className="text-foreground">Run commands from Nexul</span>
      </label>
      <p className="text-xs text-muted-foreground">
        A runner takes them only once it was installed with <span className="font-mono">--allow-shell</span>, or{" "}
        <span className="font-mono">nexul shell on &lt;runner&gt;</span> ran on the machine. Every command is recorded.
      </p>
      {on && runners.length > 0 && (
        <div className="flex items-center gap-2">
          {runners.length > 1 && (
            <Select value={runnerId} onValueChange={setPicked}>
              <SelectTrigger aria-label="Runner" className="w-36 shrink-0 font-mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {runners.map((r) => (
                  <SelectItem key={r.id} value={r.id} className="font-mono">
                    {r.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <ActionButton
            {...gate}
            type="button"
            variant="outline"
            size="sm"
            loading={openTerminal.isPending}
            onClick={() => openTerminal.mutate(runnerId, { onSuccess: setTerminal })}
          >
            <SquareTerminalIcon aria-hidden /> Open terminal
          </ActionButton>
        </div>
      )}
      {isPending && <LoadingDisplay label="Loading commands" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't list the commands." className="p-3" />}
      {commands && commands.length === 0 && (
        <EmptyRow flush className="text-xs">
          No commands have run here.
        </EmptyRow>
      )}
      {commands && commands.length > 0 && (
        <ul className="divide-y divide-border">
          {commands.map((command) => (
            <ComputerCommandRow key={command.id} command={command} named onOutput={(c) => setShown({ command: c, open: true })} />
          ))}
        </ul>
      )}
      {shown.command && (
        <ComputerCommandSheet command={shown.command} open={shown.open} onOpenChange={(open) => setShown((prev) => ({ ...prev, open }))} />
      )}
      {terminal && (
        <Suspense fallback={null}>
          <LazyTerminalDialog title={runnerName} socketPath={terminalSocketPath(terminal)} onClose={() => setTerminal(null)} />
        </Suspense>
      )}
    </div>
  );
};
