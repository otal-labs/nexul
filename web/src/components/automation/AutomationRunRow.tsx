import { AutomationRunOutcomeBadge } from "@/components/automation/AutomationRunOutcomeBadge";
import type { AutomationRun } from "@/models/AutomationRun";
import { formatDurationMs, formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationRunRowProps {
  run: AutomationRun;
  onSelect: (runId: string) => void;
}

export const AutomationRunRow = ({ run, onSelect }: AutomationRunRowProps) => (
  <li>
    <button
      type="button"
      onClick={() => onSelect(run.id)}
      className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <AutomationRunOutcomeBadge outcome={run.outcome} />
      <span className="min-w-0 flex-1 truncate font-mono text-sm">{run.event_topic}</span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
        {formatDurationMs(run.duration_ms)}
      </span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground">{formatRelativeTime(run.started_at)}</span>
    </button>
  </li>
);
