import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { useFetchSharedRuns } from "@/hooks/SharedRunHooks";
import { formatAge } from "@/utils/TimeUtility";

const runTone = (state: string): SettingsStatusTone => {
  if (state === "done") return "success";
  if (state === "failed") return "destructive";
  if (state === "starting" || state === "running" || state === "waiting") return "warning";
  return "muted";
};

interface ComputerSharedRunsProps {
  computerId: string;
}

// The play runs people the computer is shared with ran on it: who, which play, how it stands, never what it did.
export const ComputerSharedRuns = ({ computerId }: ComputerSharedRunsProps) => {
  const { data: runs, error } = useFetchSharedRuns(computerId);
  if (!error && (!runs || runs.length === 0)) return null;
  return (
    <ComputerFact label="Runs by others">
      {error && <ErrorDisplay error={error} title="Couldn't list the runs." className="p-3" />}
      {runs && runs.length > 0 && (
        <ul className="divide-y divide-border">
          {runs.map((run) => (
            <li key={run.trail_id} className="flex min-w-0 items-center gap-x-3 py-1 text-xs">
              <span className="max-w-40 shrink-0 truncate text-foreground">{run.starter_name}</span>
              <span className="min-w-0 flex-1 truncate text-muted-foreground">{run.play}</span>
              <SettingsStatus tone={runTone(run.state)} className="shrink-0">
                {run.state}
              </SettingsStatus>
              <span className="shrink-0 font-mono text-muted-foreground tabular-nums">{formatAge(run.started_at)}</span>
            </li>
          ))}
        </ul>
      )}
    </ComputerFact>
  );
};
