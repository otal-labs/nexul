import { ScrollTextIcon, SquareIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useCancelComputerCommand } from "@/hooks/ComputerCommandHooks";
import { commandLabel, commandLive, commandTone, type ComputerCommand } from "@/models/ComputerCommand";
import { formatAge } from "@/utils/TimeUtility";

interface ComputerCommandRowProps {
  command: ComputerCommand;
  onOutput: (command: ComputerCommand) => void;
}

export const ComputerCommandRow = ({ command, onOutput }: ComputerCommandRowProps) => {
  const cancel = useCancelComputerCommand();
  return (
    <li className="flex min-w-0 items-center gap-x-3 py-1">
      <SettingsStatus tone={commandTone(command)} className="w-20 shrink-0">
        {commandLabel(command)}
      </SettingsStatus>
      <span className="min-w-0 flex-1 truncate font-mono text-foreground" title={command.error || command.command}>
        {command.command}
      </span>
      {command.adapter === "mcp" && <span className="shrink-0 text-xs text-muted-foreground">agent</span>}
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{formatAge(command.created_at)}</span>
      {commandLive(command) && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={`Stop ${command.command}`}
          disabled={cancel.isPending}
          onClick={() => cancel.mutate(command.id)}
        >
          <SquareIcon aria-hidden /> Stop
        </Button>
      )}
      <Button type="button" variant="ghost" size="sm" aria-label={`Output of ${command.command}`} onClick={() => onOutput(command)}>
        <ScrollTextIcon aria-hidden /> Output
      </Button>
    </li>
  );
};
