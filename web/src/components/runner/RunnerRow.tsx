import { RemoveRunnerButton } from "@/components/runner/RemoveRunnerButton";
import { RunnerStatusBadge } from "@/components/runner/RunnerStatusBadge";
import { RunnerVersionChip } from "@/components/runner/RunnerVersionChip";
import type { Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface RunnerRowProps {
  runner: Runner;
}

export const RunnerRow = ({ runner }: RunnerRowProps) => {
  const job = runner.running_job;
  return (
    <li
      className={cn(
        "transition-colors duration-150 ease-standard hover:bg-accent/40",
        !runner.connected && "opacity-60",
      )}
    >
      <div
        className="flex items-center gap-3 px-4 py-3"
      >
        <RunnerStatusBadge connected={runner.connected} />
        <div className="min-w-0 flex-1">
          <span className="block truncate font-mono font-medium" title={runner.name || runner.id}>
            {runner.name || runner.id}
          </span>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-xs text-muted-foreground">
            <span className="wrap-anywhere">{runner.id}</span>
            <span>last seen {formatRelativeTime(runner.last_seen)}</span>
            {runner.version && <RunnerVersionChip version={runner.version} />}
          </div>
        </div>
        {job && (
          <div className="max-w-[45%] truncate text-right text-sm">
            <span className="text-muted-foreground">running </span>
            <span className="font-mono text-success">{job.service || job.kind}</span>
          </div>
        )}
        {!job && <span className="shrink-0 text-sm text-muted-foreground">idle</span>}
        <RemoveRunnerButton runner={runner} />
      </div>
    </li>
  );
};
