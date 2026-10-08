import { AutomationRow } from "@/components/automation/AutomationRow";
import { DecisionsCheckRow } from "@/components/automation/DecisionsCheckRow";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

interface AutomationsFeedProps {
  automations: Automation[];
}

// The decisions check lists after the shipped defaults, before the workspace's own automations.
export const AutomationsFeed = ({ automations }: AutomationsFeedProps) => (
  <ul className="divide-y divide-border overflow-hidden rounded-md border">
    {automations
      .filter((a) => a.kind === AutomationKind.Default)
      .map((automation) => (
        <AutomationRow key={automation.id} automation={automation} />
      ))}
    <DecisionsCheckRow />
    {automations
      .filter((a) => a.kind !== AutomationKind.Default)
      .map((automation) => (
        <AutomationRow key={automation.id} automation={automation} />
      ))}
  </ul>
);
