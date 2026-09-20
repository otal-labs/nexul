import { useState } from "react";

import { AutomationRunDetailSheet } from "@/components/automation/AutomationRunDetailSheet";
import { AutomationRunRow } from "@/components/automation/AutomationRunRow";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import type { AutomationRun } from "@/models/AutomationRun";

interface AutomationRunsFeedProps {
  automationId: string;
  runs: AutomationRun[];
}

export const AutomationRunsFeed = ({ automationId, runs }: AutomationRunsFeedProps) => {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold">Run history</h2>
      {runs.length === 0 && <NoDataDisplay message="No runs yet" size="compact" />}
      {runs.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {runs.map((run) => (
            <AutomationRunRow key={run.id} run={run} onSelect={setSelectedRunId} />
          ))}
        </ul>
      )}
      <AutomationRunDetailSheet
        automationId={automationId}
        runId={selectedRunId}
        onOpenChange={(open) => !open && setSelectedRunId(null)}
      />
    </section>
  );
};
