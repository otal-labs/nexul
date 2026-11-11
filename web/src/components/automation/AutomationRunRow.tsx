import { AutomationRunOutcome } from "@/enums/Automation";
import { cn } from "@/lib/utils";
import { runOutcomes, type AutomationRun } from "@/models/AutomationRun";
import { formatClockTime, formatDurationMs, formatFullTime } from "@/utils/TimeUtility";

interface AutomationRunRowProps {
  run: AutomationRun;
  onSelect: (runId: string) => void;
}

// The outcome, its event and the time on one line, every row alike; a failure adds its error under it.
export const AutomationRunRow = ({ run, onSelect }: AutomationRunRowProps) => {
  const outcome = runOutcomes[run.outcome];
  const failed = run.outcome !== AutomationRunOutcome.Success;
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
            <span className={cn("w-16 shrink-0 text-sm", failed && "font-medium")}>{outcome.word}</span>
            <span className="min-w-0 flex-1 truncate font-mono text-row text-muted-foreground">{run.event_topic}</span>
            <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
              {formatDurationMs(run.duration_ms)} ·{" "}
              <time dateTime={run.started_at} title={formatFullTime(run.started_at)}>
                {formatClockTime(run.started_at)}
              </time>
            </span>
          </span>
          {failed && run.error && (
            <span className="block truncate font-mono text-xs leading-5 text-muted-foreground" title={run.error}>
              {run.error}
            </span>
          )}
        </span>
      </button>
    </li>
  );
};
