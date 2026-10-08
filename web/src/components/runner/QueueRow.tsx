import { cn } from "@/lib/utils";
import { entranceDelayMs } from "@/components/runner/motion";
import type { QueuedJob } from "@/models/Runner";

interface QueueRowProps {
  job: QueuedJob;
  index?: number;
  /** 1-based queue place — draining, not accumulating, so it's worth calling out. */
  position?: number;
}

// Same split as RunnerRow; the position chip is also keyed on `position` so it fades alone on change.
export const QueueRow = ({ job, index = 0, position }: QueueRowProps) => (
  <li className="transition-colors duration-150 ease-standard hover:bg-accent/40">
    <div
      className="animate-in fade-in-0 slide-in-from-bottom-1 flex items-center gap-3 px-4 py-3 duration-150 ease-out"
      style={{ animationDelay: `${entranceDelayMs(index)}ms` }}
    >
      {position !== undefined && (
        <span
          key={position}
          className="animate-in fade-in-0 shrink-0 font-mono text-xs text-muted-foreground tabular-nums duration-150 ease-standard"
          aria-label={`Queue position ${position}`}
        >
          #{position}
        </span>
      )}
      {job.service && <span className="min-w-0 flex-1 truncate text-sm font-medium">{job.service}</span>}
      <span className={cn("min-w-0 font-mono text-xs wrap-anywhere text-muted-foreground", !job.service && "flex-1")}>
        {job.id}
      </span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground">{job.kind}</span>
    </div>
  </li>
);
