import { ArrowLeft } from "lucide-react";
import { Link } from "react-router";

import { AutomationConfigStatusBadge } from "@/components/automation/AutomationConfigStatusBadge";
import { AutomationKindBadge } from "@/components/automation/AutomationKindBadge";
import { Switch } from "@/components/ui/switch";
import { useSetAutomationEnabled } from "@/hooks/AutomationHooks";
import type { Automation } from "@/models/Automation";

interface AutomationDetailHeaderProps {
  automation: Automation;
}

export const AutomationDetailHeader = ({ automation }: AutomationDetailHeaderProps) => {
  const setEnabled = useSetAutomationEnabled();

  return (
    <div className="space-y-3 border-b border-border pb-6">
      <Link
        to="/automations"
        className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Automations
      </Link>
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-mono text-xs text-muted-foreground">{automation.id}</span>
        <AutomationKindBadge kind={automation.kind} />
        <AutomationConfigStatusBadge automation={automation} />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight sm:text-[28px]">{automation.name}</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{automation.description}</p>
        </div>
        <label className="flex shrink-0 items-center gap-2 text-sm text-muted-foreground">
          {automation.enabled ? "Enabled" : "Disabled"}
          <Switch
            aria-label={automation.enabled ? "Disable automation" : "Enable automation"}
            checked={automation.enabled}
            disabled={setEnabled.isPending}
            onCheckedChange={(checked) => setEnabled.mutate({ id: automation.id, enabled: checked })}
          />
        </label>
      </div>
    </div>
  );
};
