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
import { formatDurationMs } from "@/utils/TimeUtility";

interface AutomationRunDetailSheetProps {
  automationId: string;
  runId: string | null;
  onOpenChange: (open: boolean) => void;
}

export const AutomationRunDetailSheet = ({ automationId, runId, onOpenChange }: AutomationRunDetailSheetProps) => {
  const { data: run, error, isPending } = useFetchAutomationRun(automationId, runId ?? undefined);

  return (
    <Sheet open={runId != null} onOpenChange={onOpenChange}>
      <SheetContent className="w-full gap-6 overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Run detail</SheetTitle>
          <SheetDescription>{run?.event_topic ?? "Loading run…"}</SheetDescription>
        </SheetHeader>
        <div className="space-y-4 px-4 pb-4">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          {run && (
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-3 text-sm">
                <AutomationRunOutcomeBadge outcome={run.outcome} />
                <span className="font-mono text-muted-foreground tabular-nums">
                  {formatDurationMs(run.duration_ms)}
                </span>
                <span className="font-mono text-xs text-muted-foreground">{new Date(run.started_at).toLocaleString()}</span>
              </div>
              {run.error && <p className="text-sm text-destructive">{run.error}</p>}
              <AutomationRunLogView logs={run.logs} />
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
};
