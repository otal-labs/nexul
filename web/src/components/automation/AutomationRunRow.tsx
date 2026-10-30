import { CheckCircle2, Skull, XCircle, type LucideIcon } from "lucide-react";

import { AutomationRunOutcome } from "@/enums/Automation";
import { cn } from "@/lib/utils";
import type { AutomationRun } from "@/models/AutomationRun";
import { formatClockTime, formatDurationMs, formatFullTime } from "@/utils/TimeUtility";

const outcomes: Record<AutomationRunOutcome, { icon: LucideIcon; color: string; word: string }> = {
  [AutomationRunOutcome.Success]: { icon: CheckCircle2, color: "text-success", word: "Success" },
  [AutomationRunOutcome.Failure]: { icon: XCircle, color: "text-destructive", word: "Failure" },
  [AutomationRunOutcome.Crash]: { icon: Skull, color: "text-destructive", word: "Crash" },
};

interface AutomationRunRowProps {
  run: AutomationRun;
  onSelect: (runId: string) => void;
}

// Two lines: the outcome and its event, then the error when it failed, else how long it took and the clock time, since the day heading above names the day.
export const AutomationRunRow = ({ run, onSelect }: AutomationRunRowProps) => {
  const outcome = outcomes[run.outcome];
  const failed = run.outcome !== AutomationRunOutcome.Success;
  const when = (
    <>
      {formatDurationMs(run.duration_ms)} · <time dateTime={run.started_at} title={formatFullTime(run.started_at)}>{formatClockTime(run.started_at)}</time>
    </>
  );
  const Icon = outcome.icon;

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(run.id)}
        className="flex w-full items-start gap-3 px-4 py-2.5 text-left transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <Icon className={cn("mt-0.5 size-4 shrink-0", outcome.color)} aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-baseline gap-2 leading-5">
            <span className={cn("shrink-0 text-sm", failed && "font-medium")}>{outcome.word}</span>
            <span className="min-w-0 flex-1 truncate font-mono text-[13px] text-muted-foreground">{run.event_topic}</span>
            {failed && <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{when}</span>}
          </span>
          {failed && (
            <span className="block truncate font-mono text-xs leading-5 text-muted-foreground" title={run.error}>
              {run.error || "No error message"}
            </span>
          )}
          {!failed && <span className="block font-mono text-xs leading-5 text-muted-foreground tabular-nums">{when}</span>}
        </span>
      </button>
    </li>
  );
};
