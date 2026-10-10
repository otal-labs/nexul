import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { useFetchAutoPlayLimits, useSetAutoPlayDailyCap } from "@/hooks/PlayHooks";
import { MAX_DAILY_CAP, MIN_DAILY_CAP } from "@/models/AutoPlay";

const CAP_OPTIONS = Array.from({ length: MAX_DAILY_CAP - MIN_DAILY_CAP + 1 }, (_, i) => {
  const value = String(MIN_DAILY_CAP + i);
  return { value, label: value };
});

interface AutoPlayCapLineProps {
  workspaceId: string;
  canWrite: boolean;
}

// The workspace's loop guard, shared by every play; picking a number applies it at once, like any setting without a Save.
export const AutoPlayCapLine = ({ workspaceId, canWrite }: AutoPlayCapLineProps) => {
  const { data: limits } = useFetchAutoPlayLimits(workspaceId);
  const setCap = useSetAutoPlayDailyCap(workspaceId);
  const cap = limits?.daily_cap_per_ticket;

  return (
    cap !== undefined && (
      <p className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
        Each ticket runs at most{" "}
        {canWrite && (
          <InlineSelect
            label="Auto plays per ticket a day"
            value={String(cap)}
            options={CAP_OPTIONS}
            onChange={(value) => setCap.mutate(Number(value))}
            disabled={setCap.isPending}
            className="h-7 px-2 font-mono text-xs tabular-nums"
          />
        )}
        {!canWrite && <span className="font-mono tabular-nums">{cap}</span>} auto plays a day, across every play.
      </p>
    )
  );
};
