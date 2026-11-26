import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { AutomationRunRow } from "@/components/automation/AutomationRunRow";
import { collapseRepeatedRuns, type AutomationRunDay } from "@/utils/AutomationRunUtility";

interface AutomationRunDaySectionProps {
  day: AutomationRunDay;
  onSelect: (runId: string) => void;
}

export const AutomationRunDaySection = ({ day, onSelect }: AutomationRunDaySectionProps) => (
  <section aria-label={day.label}>
    <Microheader className="px-4 pt-5 pb-2">{day.label}</Microheader>
    <EnterList className="divide-y divide-border border-b border-border">
      {collapseRepeatedRuns(day.runs).map(({ run, count }) => (
        <AutomationRunRow key={run.id} run={run} count={count} onSelect={onSelect} />
      ))}
    </EnterList>
  </section>
);
