import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { AutomationGroupSection } from "@/components/automation/AutomationGroupSection";
import { AutomationsSummary } from "@/components/automation/AutomationsSummary";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useOfflineAutomationHostIds } from "@/hooks/AutomationHostHooks";
import { AutomationState, type Automation } from "@/models/Automation";
import { automationGroups, automationListItems } from "@/utils/AutomationListUtility";

type Filter = "all" | typeof AutomationState.Failing | typeof AutomationState.Attention | typeof AutomationState.Off;

const nothingShown: Record<Filter, string> = {
  all: "No automations yet.",
  [AutomationState.Failing]: "Nothing is failing.",
  [AutomationState.Attention]: "Nothing needs attention.",
  [AutomationState.Off]: "Every automation is on.",
};

interface AutomationsFeedProps {
  automations: Automation[];
}

// Grouped Default and Custom, each most urgent first: failing, needs attention, on, off.
export const AutomationsFeed = ({ automations }: AutomationsFeedProps) => {
  const offlineHostIds = useOfflineAutomationHostIds();
  const [filter, setFilter] = useState<Filter>("all");
  const items = automationListItems(automations, offlineHostIds);
  const shown = items.filter((item) => filter === "all" || item.state === filter);

  return (
    <div className="space-y-4">
      <AutomationsSummary items={items} />
      <div className="flex justify-end">
        <ToggleGroup
          type="single"
          variant="segmented"
          size="xs"
          aria-label="Show automations"
          value={filter}
          onValueChange={(value) => value && setFilter(value as Filter)}
        >
          <ToggleGroupItem value="all">All</ToggleGroupItem>
          <ToggleGroupItem value={AutomationState.Failing}>Failing</ToggleGroupItem>
          <ToggleGroupItem value={AutomationState.Attention}>Needs attention</ToggleGroupItem>
          <ToggleGroupItem value={AutomationState.Off}>Off</ToggleGroupItem>
        </ToggleGroup>
      </div>
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
