import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { LimitRow } from "@/components/autoplay/LimitRow";
import { useSetAutoPlayLimits } from "@/hooks/PlayHooks";
import { MAX_DAILY_CAP, MIN_DAILY_CAP } from "@/models/AutoPlay";

const CAP_OPTIONS = Array.from({ length: MAX_DAILY_CAP - MIN_DAILY_CAP + 1 }, (_, i) => {
  const value = String(MIN_DAILY_CAP + i);
  return { value, label: value };
});

interface DailyCapRowProps {
  workspaceId: string;
  cap: number;
  canWrite: boolean;
}

// The workspace's loop guard, shared by every play; a pick applies at once and shows while it saves.
export const DailyCapRow = ({ workspaceId, cap, canWrite }: DailyCapRowProps) => {
  const setCap = useSetAutoPlayLimits(workspaceId);
  const shown = setCap.isPending ? (setCap.variables?.daily_cap_per_ticket ?? cap) : cap;

  return (
    <LimitRow label="Auto plays per ticket a day" hint="Shared by every play in the workspace">
      {canWrite && (
        <InlineSelect
          label="Auto plays per ticket a day"
          value={String(shown)}
          options={CAP_OPTIONS}
          onChange={(value) => setCap.mutate({ daily_cap_per_ticket: Number(value) })}
          disabled={setCap.isPending}
          className="h-7 px-2 font-mono text-xs tabular-nums"
        />
      )}
      {!canWrite && <span className="font-mono text-xs tabular-nums">{cap}</span>}
    </LimitRow>
  );
};
