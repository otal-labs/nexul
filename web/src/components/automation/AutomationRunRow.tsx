import { AutomationRunOutcome } from "@/enums/Automation";
import { cn } from "@/lib/utils";
import { runOutcomes, type AutomationRun } from "@/models/AutomationRun";
import { automationRunSummary } from "@/utils/AutomationRunUtility";
import { formatClockTime, formatDurationMs, formatFullTime } from "@/utils/TimeUtility";

interface AutomationRunRowProps {
  run: AutomationRun;
  count?: number;
  onSelect: (runId: string) => void;
}

// The outcome's mark, what the run last reported in the sans with its event as muted mono, and how many identical runs it stands for; only a failure writes its outcome, leading its error under it.
export const AutomationRunRow = ({ run, count = 1, onSelect }: AutomationRunRowProps) => {
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
            {summary?.message && <span className="min-w-0 shrink truncate text-row text-foreground" title={`${summary.message} ${summary.fields}`.trim()}>{summary.message}</span>}
            {summary?.fields && <span className="min-w-0 shrink-[2] truncate font-mono text-xs text-muted-foreground">{summary.fields}</span>}
            <span className={cn("min-w-0 truncate font-mono", summary?.message ? "text-xs text-muted-foreground" : "text-row text-foreground")}>{run.event_topic}</span>
            <span className="flex-1" />
            {count > 1 && <span className="shrink-0 text-xs text-muted-foreground tabular-nums">{count} runs</span>}
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
