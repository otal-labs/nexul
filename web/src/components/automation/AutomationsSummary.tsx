import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { AutomationState } from "@/models/Automation";
import type { AutomationListItem } from "@/utils/AutomationListUtility";

interface StatProps {
  label: string;
  value: number;
  suffix?: string;
  className?: string;
}

const Stat = ({ label, value, suffix, className }: StatProps) => (
  <div className="min-w-0 space-y-1.5 px-5 py-4">
    <p className={microheaderClass}>{label}</p>
    <p className={cn("flex items-baseline gap-1.5 font-mono text-xl font-semibold tabular-nums", className)}>
      {value}
      {suffix && <span className="text-sm font-normal text-muted-foreground">{suffix}</span>}
    </p>
  </div>
);

interface AutomationsSummaryProps {
  items: AutomationListItem[];
}

// Counts automations, never runs: Failing and Needs attention are the same sets the filter and the row chips name.
export const AutomationsSummary = ({ items }: AutomationsSummaryProps) => {
  const count = (state: AutomationState) => items.filter((item) => item.state === state).length;
  const failing = count(AutomationState.Failing);

  return (
    <div role="group" aria-label="Automations summary" className="grid grid-cols-3 divide-x divide-border rounded-lg bg-surface-2 ring-1 ring-border ring-inset">
      <Stat label="On" value={items.length - count(AutomationState.Off)} suffix={`of ${items.length}`} />
      <Stat label="Failing" value={failing} className={cn(failing > 0 && "text-destructive")} />
      <Stat label="Needs attention" value={count(AutomationState.Attention)} />
    </div>
  );
};
