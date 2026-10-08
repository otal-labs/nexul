import { cn } from "@/lib/utils";
import type { QueuedJob } from "@/models/Runner";

interface QueueRowProps {
  job: QueuedJob;
  /** 1-based queue place — draining, not accumulating, so it's worth calling out. */
  position?: number;
}

// The position chip is keyed on `position` so it fades alone on change.
export const QueueRow = ({ job, position }: QueueRowProps) => (
  <li className="transition-colors duration-150 ease-standard hover:bg-accent/40">
    <div
      className="flex items-center gap-3 px-4 py-3"
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
