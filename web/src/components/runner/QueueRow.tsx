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
          className="animate-in fade-in-0 shrink-0 rounded-full bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground duration-150 ease-standard"
          aria-label={`Queue position ${position}`}
        >
          #{position}
        </span>
      )}
      {job.service && <span className="flex-1 truncate text-sm font-medium">{job.service}</span>}
      <span className={cn("font-mono text-xs text-muted-foreground", !job.service && "flex-1")}>
        {job.id}
      </span>
      <span className="shrink-0 rounded-full border border-border bg-surface-2 px-2 py-0.5 font-mono text-xs text-muted-foreground">
        {job.kind}
      </span>
    </div>
  </li>
);
