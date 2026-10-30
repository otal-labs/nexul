import { AutomationRunLogView } from "@/components/automation/AutomationRunLogView";
import { AutomationRunOutcomeBadge } from "@/components/automation/AutomationRunOutcomeBadge";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useFetchAutomationRun } from "@/hooks/AutomationRunHooks";
import { formatDurationMs, formatVersionTime } from "@/utils/TimeUtility";

interface AutomationRunDetailSheetProps {
  automationId: string;
  runId: string | null;
  onOpenChange: (open: boolean) => void;
}

export const AutomationRunDetailSheet = ({ automationId, runId, onOpenChange }: AutomationRunDetailSheetProps) => {
  const { data: run, error, isPending } = useFetchAutomationRun(automationId, runId ?? undefined);

  return (
    <Sheet open={runId != null} onOpenChange={onOpenChange}>
      <SheetContent className="w-full gap-2 overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="font-mono text-base break-all">{run?.event_topic ?? "Run"}</SheetTitle>
          <SheetDescription className="font-mono text-xs break-all">{runId}</SheetDescription>
        </SheetHeader>
        <div className="space-y-4 px-4 pb-4">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          {run && (
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3 text-sm">
                <AutomationRunOutcomeBadge outcome={run.outcome} />
                <span className="font-mono text-xs text-muted-foreground tabular-nums">{formatDurationMs(run.duration_ms)}</span>
                <span className="font-mono text-xs text-muted-foreground">{formatVersionTime(run.started_at)}</span>
              </div>
              {run.error && (
                <p className="rounded-r-md border-l-2 border-destructive bg-destructive/5 px-3 py-2 font-mono text-xs whitespace-pre-wrap break-words">{run.error}</p>
              )}
              <AutomationRunLogView logs={run.logs} />
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
};
