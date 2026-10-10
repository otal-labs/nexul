import { EnterList } from "@/components/EnterList";
import { AutomationRow } from "@/components/automation/AutomationRow";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

interface AutomationsFeedProps {
  automations: Automation[];
}

// The shipped defaults list before the workspace's own automations.
export const AutomationsFeed = ({ automations }: AutomationsFeedProps) => (
  <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
    {automations
      .filter((a) => a.kind === AutomationKind.Default)
      .map((automation) => (
        <AutomationRow key={automation.id} automation={automation} />
      ))}
    {automations
      .filter((a) => a.kind !== AutomationKind.Default)
      .map((automation) => (
        <AutomationRow key={automation.id} automation={automation} />
      ))}
  </EnterList>
);
