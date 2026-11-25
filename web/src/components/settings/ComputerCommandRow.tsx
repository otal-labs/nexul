import { SquareIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useCancelComputerCommand } from "@/hooks/ComputerCommandHooks";
import { commandLabel, commandLive, commandTone, type ComputerCommand } from "@/models/ComputerCommand";
import { formatAge } from "@/utils/TimeUtility";

interface ComputerCommandRowProps {
  command: ComputerCommand;
  onOutput: (command: ComputerCommand) => void;
}

// One command: the row opens its output, and a live one keeps its Stop beside it.
export const ComputerCommandRow = ({ command, onOutput }: ComputerCommandRowProps) => {
  const cancel = useCancelComputerCommand();
  return (
    <li className="flex min-w-0 items-center gap-1 py-0.5">
      <button
        type="button"
        aria-label={`Output of ${command.command}`}
        title={command.error || command.command}
        onClick={() => onOutput(command)}
        className="flex min-w-0 flex-1 items-center gap-x-3 rounded-md px-1.5 py-1 text-left transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <SettingsStatus tone={commandTone(command)} className="w-20 shrink-0">
          {commandLabel(command)}
        </SettingsStatus>
        <span className="min-w-0 flex-1 truncate font-mono text-foreground">{command.command}</span>
        {command.requester_id !== command.owner_id && (
          <span className="max-w-32 shrink-0 truncate text-xs text-muted-foreground">{command.requester_name}</span>
        )}
        {command.adapter === "mcp" && <span className="shrink-0 text-xs text-muted-foreground">agent</span>}
        <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{formatAge(command.created_at)}</span>
      </button>
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
    </li>
  );
};
