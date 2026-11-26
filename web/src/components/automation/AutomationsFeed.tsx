import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { AutomationGroupSection } from "@/components/automation/AutomationGroupSection";
import { AutomationListBar } from "@/components/automation/AutomationListBar";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { useOfflineAutomationHostIds } from "@/hooks/AutomationHostHooks";
import { AutomationState, type Automation } from "@/models/Automation";
import { automationGroups, automationListItems, automationNeedsAttention, type AutomationListItem } from "@/utils/AutomationListUtility";

type Filter = "all" | typeof AutomationState.Failing | typeof AutomationState.Attention | typeof AutomationState.Off;

const filters: { value: Filter; label: string }[] = [
  { value: "all", label: "All" },
  { value: AutomationState.Failing, label: "Failing" },
  { value: AutomationState.Attention, label: "Needs attention" },
  { value: AutomationState.Off, label: "Off" },
];

const nothingShown: Record<Filter, string> = {
  all: "No automations yet.",
  [AutomationState.Failing]: "Nothing is failing.",
  [AutomationState.Attention]: "Nothing needs attention.",
  [AutomationState.Off]: "Every automation is on.",
};

interface AutomationsFeedProps {
  automations: Automation[];
}

// Grouped Default and Custom, each most urgent first; the filter's counts are the same sets the row chips name.
export const AutomationsFeed = ({ automations }: AutomationsFeedProps) => {
  const offlineHostIds = useOfflineAutomationHostIds();
  const [filter, setFilter] = useState<Filter>("all");
  const items = automationListItems(automations, offlineHostIds);
  const matches = (item: AutomationListItem, value: Filter) => {
    if (value === "all") return true;
    if (value === AutomationState.Attention) return automationNeedsAttention(item);
    return item.state === value;
  };
  const shown = items.filter((item) => matches(item, filter));
  const count = (value: Filter) => items.filter((item) => matches(item, value)).length;
  const on = items.length - count(AutomationState.Off);

  return (
    <div className="space-y-1">
      <AutomationListBar
        summary={
          <span className="font-mono tabular-nums">
            {items.length} {items.length === 1 ? "automation" : "automations"} · {on} on
          </span>
        }
      >
        <ToggleGroup type="single" variant="segmented" size="xs" aria-label="Show automations" value={filter} onValueChange={(value) => value && setFilter(value as Filter)}>
          {filters.map(({ value, label }) => (
            <ToggleGroupItem key={value} value={value} className="gap-1.5">
              {label}
              <span className={cn("font-mono tabular-nums", value === AutomationState.Failing && count(value) > 0 ? "text-destructive" : "text-muted-foreground")}>{count(value)}</span>
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </AutomationListBar>
      {shown.length === 0 && <EmptyRow className="border-y border-border">{nothingShown[filter]}</EmptyRow>}
      {shown.length > 0 && (
        <div>
          {automationGroups(shown).map((group) => (
            <AutomationGroupSection key={group.label} group={group} />
          ))}
        </div>
      )}
    </div>
  );
};
