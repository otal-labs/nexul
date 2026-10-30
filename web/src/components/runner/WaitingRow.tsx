import { JobMeta } from "@/components/runner/JobMeta";
import { JobName } from "@/components/runner/JobName";
import { fleetTracks } from "@/components/runner/fleetTracks";
import type { QueuedJob } from "@/models/Runner";
import { cn } from "@/lib/utils";
import type { QueueReason } from "@/utils/RunnerFleetUtility";

interface WaitingRowProps {
  job: QueuedJob;
  position: number;
  reason: QueueReason;
  className?: string;
}

// On the runner list's tracks, so the kind sits under Version and why it waits under Doing, wrapping rather than cut.
export const WaitingRow = ({ job, position, reason, className }: WaitingRowProps) => (
  <li className={cn(fleetTracks, "items-baseline gap-x-4 px-4 py-2.5 text-[13px] leading-5 transition-colors duration-150 ease-standard hover:bg-accent/40", className)}>
    <span className="flex min-w-0 items-baseline gap-1.5">
      <span className="w-3.5 shrink-0 text-right font-mono text-xs text-muted-foreground tabular-nums" aria-label={`Queue position ${position}`}>
        {position}
      </span>
      <JobName name={job.service || job.kind} />
    </span>
    <span className="font-mono text-xs text-muted-foreground">{job.kind}</span>
    <span className="@container/doing col-span-2 min-w-0 text-muted-foreground">
      {reason} on {job.target && <span className="font-mono text-xs whitespace-nowrap">{job.target}</span>}
      {!job.target && "any machine"}
      <JobMeta at={job.requested_at} by={job.triggered_by} inline />
    </span>
  </li>
);
