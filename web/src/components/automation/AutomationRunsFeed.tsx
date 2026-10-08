import { useState } from "react";

import { AutomationRunDetailSheet } from "@/components/automation/AutomationRunDetailSheet";
import { AutomationRunRow } from "@/components/automation/AutomationRunRow";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import type { AutomationRun } from "@/models/AutomationRun";

interface AutomationRunsFeedProps {
  automationId: string;
  runs: AutomationRun[];
}

export const AutomationRunsFeed = ({ automationId, runs }: AutomationRunsFeedProps) => {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);

  return (
    <SettingsCard id="runs" title="Run history" description="Every event this automation handled. Open a run to read its log.">
      {runs.length === 0 && <EmptyRow>No runs yet</EmptyRow>}
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
    </SettingsCard>
  );
};
