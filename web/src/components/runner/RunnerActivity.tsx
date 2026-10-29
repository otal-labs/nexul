import type { Runner } from "@/models/Runner";
import { formatAge } from "@/utils/TimeUtility";

interface RunnerActivityProps {
  runner: Runner;
}

// What the runner is doing, the busy dot hung in the gutter; last seen shows only offline, where it says something.
export const RunnerActivity = ({ runner }: RunnerActivityProps) => {
  const job = runner.running_job;
  const age = formatAge(runner.last_seen);
  return (
    <span className="flex min-w-0 items-baseline gap-2 text-[13px]">
      {job && (
        <>
          <span aria-hidden className="-ml-3.5 size-1.5 shrink-0 self-center animate-[status-pulse_2.4s_ease-standard_infinite] rounded-full bg-info" />
          <span className="truncate" title={`Running ${job.service || job.kind}`}>
            {job.service || job.kind}
          </span>
        </>
      )}
      {!job && runner.connected && <span className="text-muted-foreground">idle</span>}
      {!runner.connected && (
        <span className="truncate text-muted-foreground">
          offline · {age === "now" && "just now"}
          {age !== "now" && (
            <>
              <span className="font-mono text-xs tabular-nums">{age}</span> ago
            </>
          )}
        </span>
      )}
    </span>
  );
};
