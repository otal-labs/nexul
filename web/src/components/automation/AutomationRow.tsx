import { Link } from "react-router";

import { AutomationConfigStatusBadge } from "@/components/automation/AutomationConfigStatusBadge";
import { AutomationKindBadge } from "@/components/automation/AutomationKindBadge";
import { Switch } from "@/components/ui/switch";
import { useSetAutomationEnabled } from "@/hooks/AutomationHooks";
import type { Automation } from "@/models/Automation";

interface AutomationRowProps {
  automation: Automation;
}

export const AutomationRow = ({ automation }: AutomationRowProps) => {
  const setEnabled = useSetAutomationEnabled();

  return (
    <li className="flex items-center gap-3 p-4 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Link to={`/automations/${automation.id}`} className="truncate text-sm font-medium hover:underline">
            {automation.name}
          </Link>
          <AutomationKindBadge kind={automation.kind} />
          <AutomationConfigStatusBadge automation={automation} />
        </div>
        <p className="truncate text-sm text-muted-foreground">{automation.description}</p>
      </div>
      <Switch
        aria-label={automation.enabled ? `Disable ${automation.name}` : `Enable ${automation.name}`}
        checked={automation.enabled}
        disabled={setEnabled.isPending}
        onCheckedChange={(checked) => setEnabled.mutate({ id: automation.id, enabled: checked })}
      />
    </li>
  );
};
