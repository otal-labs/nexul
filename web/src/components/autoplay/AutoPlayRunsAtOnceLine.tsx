import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { useFetchAutoPlayLimits, useSetAutoPlayLimits } from "@/hooks/PlayHooks";
import { MAX_RUNS_AT_ONCE, MIN_RUNS_AT_ONCE } from "@/models/AutoPlay";

const RUNS_OPTIONS = Array.from({ length: MAX_RUNS_AT_ONCE - MIN_RUNS_AT_ONCE + 1 }, (_, i) => {
  const value = String(MIN_RUNS_AT_ONCE + i);
  return { value, label: value };
});

interface AutoPlayRunsAtOnceLineProps {
  workspaceId: string;
  canWrite: boolean;
}

// How many runs a person may have going before their next auto run waits; shared by every play, applied without a Save.
export const AutoPlayRunsAtOnceLine = ({ workspaceId, canWrite }: AutoPlayRunsAtOnceLineProps) => {
  const { data: limits } = useFetchAutoPlayLimits(workspaceId);
  const setLimits = useSetAutoPlayLimits(workspaceId);
  const runs = limits?.runs_at_once_per_person;

  return (
    runs !== undefined && (
      <p className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
        Each person has at most{" "}
        {canWrite && (
          <InlineSelect
            label="Runs at once per person"
            value={String(runs)}
            options={RUNS_OPTIONS}
            onChange={(value) => setLimits.mutate({ runs_at_once_per_person: Number(value) })}
            disabled={setLimits.isPending}
            className="h-7 px-2 font-mono text-xs tabular-nums"
          />
        )}
        {!canWrite && <span className="font-mono tabular-nums">{runs}</span>} runs going at once; more auto runs wait.
      </p>
    )
  );
};
