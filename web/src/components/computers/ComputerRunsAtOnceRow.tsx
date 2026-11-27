import { SettingsRow } from "@/components/settings/SettingsRow";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { RUNS_AT_ONCE_CHOICES, type Computer } from "@/models/Pairing";

interface ComputerRunsAtOnceRowProps {
  computer: Computer;
}

// How many runs the computer takes together (ADR 0166): its one home, and the only limit on how much runs.
export const ComputerRunsAtOnceRow = ({ computer }: ComputerRunsAtOnceRowProps) => {
  const update = useUpdateComputerSettings(computer.id);

  return (
    <SettingsRow label="Runs at once" description="How many runs this computer takes together" compact className="last:pb-4">
      <Select value={String(computer.runs_at_once)} onValueChange={(v) => update.mutate({ runs_at_once: Number(v) })}>
        <SelectTrigger aria-label={`Runs at once on ${computer.name}`} className="h-8 w-20 font-mono text-xs tabular-nums">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {RUNS_AT_ONCE_CHOICES.map((n) => (
            <SelectItem key={n} value={n} className="font-mono text-xs tabular-nums">
              {n}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </SettingsRow>
  );
};
