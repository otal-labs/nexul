import { entranceDelayMs } from "@/components/runner/motion";
import { RunnerStatusBadge } from "@/components/runner/RunnerStatusBadge";
import { RunnerVersionChip } from "@/components/runner/RunnerVersionChip";
import type { Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface RunnerRowProps {
  runner: Runner;
  index?: number;
}

// <li> owns hover, the inner <div> owns the entrance, so a refetch reusing key={runner.id} won't replay it.
export const RunnerRow = ({ runner, index = 0 }: RunnerRowProps) => {
  const job = runner.running_job;
  return (
    <li
      className={cn(
        "transition-colors duration-150 ease-standard hover:bg-accent/40",
        !runner.connected && "opacity-60",
      )}
    >
      <div
        className="animate-in fade-in-0 slide-in-from-bottom-1 flex items-center gap-3 px-4 py-3 duration-150 ease-out"
        style={{ animationDelay: `${entranceDelayMs(index)}ms` }}
      >
        <RunnerStatusBadge connected={runner.connected} />
        <div className="min-w-0 flex-1">
          <span className="block truncate font-mono font-medium">{runner.name || runner.id}</span>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1">
            <p className="truncate font-mono text-xs text-muted-foreground">
              {runner.id} · last seen {formatRelativeTime(runner.last_seen)}
            </p>
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
      </div>
    </li>
  );
};
