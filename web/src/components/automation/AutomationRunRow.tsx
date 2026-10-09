import { ChevronRightIcon } from "lucide-react";

import { AutomationRunOutcomeBadge } from "@/components/automation/AutomationRunOutcomeBadge";
import type { AutomationRun } from "@/models/AutomationRun";
import { formatDurationMs, formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationRunRowProps {
  run: AutomationRun;
  onSelect: (runId: string) => void;
}

// Fixed columns, so the event never shifts with the width of Success, Failure or Crash.
export const AutomationRunRow = ({ run, onSelect }: AutomationRunRowProps) => (
  <li>
    <button
      type="button"
      onClick={() => onSelect(run.id)}
      className="group grid w-full grid-cols-[5rem_minmax(0,1fr)_4.5rem_4.5rem_1rem] items-center gap-x-3 px-4 py-2.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <AutomationRunOutcomeBadge outcome={run.outcome} />
      <span className="truncate font-mono text-sm">{run.event_topic}</span>
      <span className="text-right font-mono text-xs text-muted-foreground tabular-nums">{formatDurationMs(run.duration_ms)}</span>
      <span className="text-right font-mono text-xs text-muted-foreground tabular-nums" title={run.started_at}>
        {formatRelativeTime(run.started_at)}
      </span>
      <ChevronRightIcon
        className="size-4 text-muted-foreground/50 transition-colors duration-150 ease-standard group-hover:text-muted-foreground"
        aria-hidden
      />
    </button>
  </li>
);
