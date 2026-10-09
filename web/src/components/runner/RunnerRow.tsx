import { RemoveRunnerButton } from "@/components/runner/RemoveRunnerButton";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import { RunnerVersionChip } from "@/components/runner/RunnerVersionChip";
import type { Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface RunnerRowProps {
  runner: Runner;
  /** Under a machine of the same name the row names the runner by its short id instead of repeating the header. */
  machineName?: string;
}

// Fixed columns, the dot under the machine's mark, so names, times and state line up across every machine.
export const RunnerRow = ({ runner, machineName }: RunnerRowProps) => {
  const job = runner.running_job;
  const named = runner.name && runner.name !== machineName;
  return (
    <li className="grid grid-cols-[2.25rem_minmax(0,1fr)_5.5rem_minmax(0,9rem)_2rem] items-center gap-x-3 px-4 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <RunnerStatusDot connected={runner.connected} />
      <div className="min-w-0">
        <span className={cn("block truncate font-mono text-sm", !runner.connected && "text-muted-foreground")} title={runner.id}>
          {named ? runner.name : `runner ${runner.id.slice(0, 8)}`}
        </span>
        {runner.version && <RunnerVersionChip version={runner.version} />}
      </div>
      <span className="text-right font-mono text-xs text-muted-foreground tabular-nums" title="Last seen">
        {formatRelativeTime(runner.last_seen)}
      </span>
      <span className="min-w-0 truncate text-sm">
        {job && (
          <span className="inline-flex max-w-full items-center gap-1.5" title={`Running ${job.service || job.kind}`}>
            <span aria-hidden className="size-1.5 shrink-0 animate-[status-pulse_2.4s_ease-standard_infinite] rounded-full bg-info" />
            <span className="truncate font-mono text-xs">{job.service || job.kind}</span>
          </span>
        )}
        {!job && runner.connected && <span className="text-muted-foreground">idle</span>}
        {!runner.connected && <span className="text-muted-foreground">offline</span>}
      </span>
      <RemoveRunnerButton runner={runner} />
    </li>
  );
};
