import { JobMeta } from "@/components/runner/JobMeta";
import { JobName } from "@/components/runner/JobName";
import { RunnerManagerTag } from "@/components/runner/RunnerManagerTag";
import { RunnerStatusDot } from "@/components/runner/RunnerStatusDot";
import type { Runner } from "@/models/Runner";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface RunnerActivityProps {
  runner: Runner;
}

// The runner's one state, a dot beside its word set against the version: a pulsing job, idle, or offline with when it was last seen, worded as the band words it.
export const RunnerActivity = ({ runner }: RunnerActivityProps) => {
  const job = runner.running_job;
  return (
    <span className="@container/doing flex min-w-0 items-baseline justify-end gap-1 text-row tabular-nums">
      {job && <span aria-hidden className="mr-1 size-2 shrink-0 self-center animate-[status-pulse_2.4s_ease-standard_infinite] rounded-full bg-info" />}
      {job && <JobName name={job.service || job.kind} />}
      {job && <JobMeta at={job.started_at} by={job.triggered_by} />}
      {!job && (
        <span className="mr-1 flex self-center">
          <RunnerStatusDot connected={runner.connected} />
        </span>
      )}
      {!job && runner.connected && <span className="text-muted-foreground">idle</span>}
      {!runner.connected && <span className="truncate text-muted-foreground">offline · seen {formatRelativeTime(runner.last_seen)}</span>}
      {runner.managers?.map((m) => <RunnerManagerTag key={m.manager} manager={m} className="ml-2" />)}
    </span>
  );
};
