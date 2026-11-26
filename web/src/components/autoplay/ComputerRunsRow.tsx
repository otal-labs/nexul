import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { cn } from "@/lib/utils";
import { LimitRow } from "@/components/autoplay/LimitRow";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { RUNS_AT_ONCE_CHOICES, type Computer } from "@/models/Pairing";

const RUNS_OPTIONS = RUNS_AT_ONCE_CHOICES.map((n) => ({ value: n, label: n }));

// One component per computer because the update hook is bound to a single computer id; the same compact select as the daily cap above it.
export const ComputerRunsRow = ({ computer }: { computer: Computer }) => {
  const update = useUpdateComputerSettings(computer.id);
  const name = computer.name || "New computer";
  const shown = update.isPending ? (update.variables?.runs_at_once ?? computer.runs_at_once) : computer.runs_at_once;

  return (
    <LimitRow label={<span title={name} className={cn(!computer.name && "text-muted-foreground")}>{name}</span>}>
      <InlineSelect
        label={`Runs at once on ${name}`}
        value={String(shown)}
        options={RUNS_OPTIONS}
        onChange={(value) => update.mutate({ runs_at_once: Number(value) })}
        disabled={update.isPending}
        className="h-7 px-2 font-mono text-xs tabular-nums"
      />
    </LimitRow>
  );
};
