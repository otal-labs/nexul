import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { RUNS_AT_ONCE_CHOICES, type Computer } from "@/models/Pairing";

const OPTIONS = RUNS_AT_ONCE_CHOICES.map((value) => ({ value, label: value }));

interface ComputerRunsSelectProps {
  computer: Computer;
  lead: string;
  tail: string;
  /** Opens the last item with "and", kept on the item's line so it never ends one alone. */
  and?: boolean;
}

// One component per computer because the update hook is bound to a single computer id.
export const ComputerRunsSelect = ({ computer, lead, tail, and = false }: ComputerRunsSelectProps) => {
  const update = useUpdateComputerSettings(computer.id);

  return (
    <>
      <span className="inline-flex items-center gap-1 whitespace-nowrap align-middle">
        {and && "and"}
        <InlineSelect
          label={`Runs at once on ${computer.name}`}
          value={String(computer.runs_at_once)}
          options={OPTIONS}
          onChange={(value) => update.mutate({ runs_at_once: Number(value) })}
          disabled={update.isPending}
          className="h-7 px-2 font-mono text-xs tabular-nums"
        />
        {lead}
        <span>
          <span className="inline-block max-w-56 truncate align-bottom lg:max-w-40 xl:max-w-96 text-foreground" title={computer.name}>
            {computer.name}
          </span>
          {tail}
        </span>
      </span>{" "}
    </>
  );
};
