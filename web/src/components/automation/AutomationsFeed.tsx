import { AutomationRow } from "@/components/automation/AutomationRow";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import type { Automation } from "@/models/Automation";

interface AutomationsFeedProps {
  automations: Automation[];
}

export const AutomationsFeed = ({ automations }: AutomationsFeedProps) => (
  <div className="space-y-4">
    {automations.length === 0 && <NoDataDisplay message="No automations yet" />}
    {automations.length > 0 && (
      <ul className="divide-y divide-border overflow-hidden rounded-md border">
        {automations.map((automation) => (
          <AutomationRow key={automation.id} automation={automation} />
        ))}
      </ul>
    )}
  </div>
);
