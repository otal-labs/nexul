import { AutomationRunOutcome } from "@/enums/Automation";
import { cn } from "@/lib/utils";
import { runOutcomes, type AutomationRun } from "@/models/AutomationRun";
import { automationRunSummary } from "@/utils/AutomationRunUtility";
import { formatClockTime, formatDurationMs, formatFullTime } from "@/utils/TimeUtility";

interface AutomationRunRowProps {
  run: AutomationRun;
  onSelect: (runId: string) => void;
}

// The outcome's mark, its event and what the run last reported on one line; only a failure writes its outcome, leading its error under it.
export const AutomationRunRow = ({ run, onSelect }: AutomationRunRowProps) => {
  const outcome = runOutcomes[run.outcome];
  const failed = run.outcome !== AutomationRunOutcome.Success;
  const Icon = outcome.icon;
  const summary = automationRunSummary(run.logs);

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(run.id)}
        className="flex w-full items-start gap-3 px-4 py-2 text-left transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <Icon className={cn("mt-0.5 size-4 shrink-0", outcome.color)} aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-baseline gap-2 leading-5">
            {!failed && <span className="sr-only">{outcome.word}</span>}
            <span className="max-w-1/3 min-w-0 shrink-0 truncate font-mono text-row text-foreground">{run.event_topic}</span>
            <span className="min-w-0 flex-1 truncate text-row" title={summary ? `${summary.message} ${summary.fields}`.trim() : undefined}>
              {summary?.message}
              {summary?.fields && <span className="ml-2 font-mono text-xs text-muted-foreground">{summary.fields}</span>}
            </span>
            <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
              {formatDurationMs(run.duration_ms)} ·{" "}
              <time dateTime={run.started_at} title={formatFullTime(run.started_at)}>
                {formatClockTime(run.started_at)}
              </time>
            </span>
          </span>
          {failed && (
            <span className="flex min-w-0 items-baseline gap-2 text-xs leading-5" title={run.error}>
              <span className="shrink-0 font-medium">{outcome.word}</span>
              {run.error && <span className="truncate font-mono text-muted-foreground">{run.error}</span>}
            </span>
          )}
        </span>
      </button>
    </li>
  );
};
