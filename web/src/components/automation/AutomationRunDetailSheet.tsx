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
          <SheetDescription className="flex flex-wrap items-center gap-3 text-sm">
            {run && <AutomationRunOutcomeBadge outcome={run.outcome} />}
            {run && <span className="font-mono text-xs tabular-nums">{formatDurationMs(run.duration_ms)}</span>}
            {run && <span className="font-mono text-xs">{formatVersionTime(run.started_at)}</span>}
          </SheetDescription>
        </SheetHeader>
        <div className="space-y-4 px-4 pb-4">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          {run && (
            <div className="space-y-4">
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
