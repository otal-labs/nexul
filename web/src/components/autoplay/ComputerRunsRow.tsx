import { LimitRow } from "@/components/autoplay/LimitRow";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { RUNS_AT_ONCE_CHOICES, type Computer } from "@/models/Pairing";

// One component per computer because the update hook is bound to a single computer id; the same control as the computer's settings.
export const ComputerRunsRow = ({ computer }: { computer: Computer }) => {
  const update = useUpdateComputerSettings(computer.id);
  const shown = update.isPending ? (update.variables?.runs_at_once ?? computer.runs_at_once) : computer.runs_at_once;

  return (
    <LimitRow label={<span title={computer.name}>{computer.name}</span>}>
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        value={String(shown)}
        // Radix reports "" when the chosen item is clicked again; keep the current count.
        onValueChange={(v) => v && update.mutate({ runs_at_once: Number(v) })}
        disabled={update.isPending}
        aria-label={`Runs at once on ${computer.name}`}
      >
        {RUNS_AT_ONCE_CHOICES.map((n) => (
          <ToggleGroupItem key={n} value={n} aria-label={n} className="h-6 w-8 px-0 font-mono tabular-nums">
            {n}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
    </LimitRow>
  );
};
