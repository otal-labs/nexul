import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AutomationRunDaySection } from "@/components/automation/AutomationRunDaySection";
import { AutomationRunDetailSheet } from "@/components/automation/AutomationRunDetailSheet";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useFetchAutomationRuns } from "@/hooks/AutomationRunHooks";
import type { Automation } from "@/models/Automation";
import { automationRunDays, automationRunsLine } from "@/utils/AutomationRunUtility";

type Show = "all" | "failures";

// What the first run waits for, from the events the automation's code declares.
const noRunsYet = (subscriptions: readonly string[]) => {
  if (subscriptions.length === 0) return "No runs yet. It reacts to no events yet.";
  if (subscriptions.length === 1) return `No runs yet. It runs when ${subscriptions[0]} arrives.`;
  return `No runs yet. It runs when one of its ${subscriptions.length} events arrives.`;
};

interface AutomationRunsFeedProps {
  automation: Automation;
}

export const AutomationRunsFeed = ({ automation }: AutomationRunsFeedProps) => {
  const [show, setShow] = useState<Show>("all");
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const failedOnly = show === "failures";
  const { data: runs, error, isPending } = useFetchAutomationRuns(automation.id, failedOnly);

  return (
    <div className="space-y-1">
      <div className="flex h-8 items-center justify-between gap-3 px-4">
        <p className="font-mono text-xs text-muted-foreground tabular-nums">{runs && runs.length > 0 && automationRunsLine(runs, failedOnly)}</p>
        <ToggleGroup type="single" variant="segmented" size="xs" aria-label="Show runs" value={show} onValueChange={(value) => value && setShow(value as Show)}>
          <ToggleGroupItem value="all">All</ToggleGroupItem>
          <ToggleGroupItem value="failures">Failures</ToggleGroupItem>
        </ToggleGroup>
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {runs && runs.length === 0 && !failedOnly && <EmptyRow className="border-y border-border">{noRunsYet(automation.subscriptions)}</EmptyRow>}
      {runs && runs.length === 0 && failedOnly && <EmptyRow className="border-y border-border">No failures in the last 30 days.</EmptyRow>}
      {runs && runs.length > 0 && (
        <div>
          {automationRunDays(runs).map((day) => (
            <AutomationRunDaySection key={day.label} day={day} onSelect={setSelectedRunId} />
          ))}
        </div>
      )}
      <AutomationRunDetailSheet automationId={automation.id} runId={selectedRunId} onOpenChange={(open) => !open && setSelectedRunId(null)} />
    </div>
  );
};
