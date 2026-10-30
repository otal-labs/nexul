import { JobName } from "@/components/runner/JobName";
import type { Runner } from "@/models/Runner";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface RunnerActivityProps {
  runner: Runner;
}

// What the runner is doing, its busy dot hung in the gutter; last seen shows only offline, worded as the band words it.
export const RunnerActivity = ({ runner }: RunnerActivityProps) => {
  const job = runner.running_job;
  return (
    <span className="flex min-w-0 items-baseline gap-2 text-[13px] tabular-nums">
      {job && <span aria-hidden className="-ml-4 size-2 shrink-0 self-center animate-[status-pulse_2.4s_ease-standard_infinite] rounded-full bg-info" />}
      {job && <JobName name={job.service || job.kind} />}
      {!job && runner.connected && <span className="text-muted-foreground">idle</span>}
      {!runner.connected && <span className="truncate text-muted-foreground">offline · seen {formatRelativeTime(runner.last_seen)}</span>}
    </span>
  );
};
