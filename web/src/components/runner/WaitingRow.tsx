import type { QueuedJob } from "@/models/Runner";
import type { QueueReason } from "@/utils/RunnerFleetUtility";

interface WaitingRowProps {
  job: QueuedJob;
  position: number;
  reason: QueueReason;
}

// Below 44rem the machine folds into the reason; from there it and the kind get columns of their own.
export const WaitingRow = ({ job, position, reason }: WaitingRowProps) => {
  const machine = job.target || "any machine";
  return (
    <li className="grid grid-cols-[1.5rem_minmax(14rem,1fr)_minmax(0,1fr)] items-baseline gap-3 px-4 py-2.5 leading-5 transition-colors duration-150 ease-standard hover:bg-accent/40 @[44rem]:grid-cols-[1.5rem_minmax(14rem,1fr)_12rem_8rem_3.5rem]">
      <span className="font-mono text-xs text-muted-foreground tabular-nums" aria-label={`Queue position ${position}`}>
        #{position}
      </span>
      <span className="truncate text-[13px]">{job.service || job.kind}</span>
      <span className="hidden truncate font-mono text-xs text-muted-foreground @[44rem]:block">{machine}</span>
      <span className="flex min-w-0 items-baseline text-[13px] text-muted-foreground">
        <span className="shrink-0">{reason}</span>
        <span className="flex min-w-0 items-baseline @[44rem]:hidden">
          <span className="shrink-0">&nbsp;on&nbsp;</span>
          <span className="truncate font-mono text-xs">{machine}</span>
        </span>
      </span>
      <span className="hidden text-right font-mono text-xs text-muted-foreground @[44rem]:block">{job.kind}</span>
    </li>
  );
};
