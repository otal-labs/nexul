import { AutomationConfigStatusBadge } from "@/components/automation/AutomationConfigStatusBadge";
import { AutomationKindBadge } from "@/components/automation/AutomationKindBadge";
import { PageHeader, pageTitleClass } from "@/components/PageHeader";
import { Switch } from "@/components/ui/switch";
import { useSetAutomationEnabled } from "@/hooks/AutomationHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Automation } from "@/models/Automation";

interface AutomationDetailHeaderProps {
  automation: Automation;
}

export const AutomationDetailHeader = ({ automation }: AutomationDetailHeaderProps) => {
  const setEnabled = useSetAutomationEnabled();
  const wsPath = useWorkspacePath();
  const workspaceCrumb = useWorkspaceCrumb();

  return (
    <PageHeader
      crumbs={[workspaceCrumb, { label: "Automations", to: wsPath("/automations") }]}
      title={
        <>
          <h1 className={pageTitleClass}>{automation.name}</h1>
          {automation.description && (
            <p className="mt-1.5 max-w-2xl text-sm text-pretty text-muted-foreground">{automation.description}</p>
          )}
        </>
      }
      meta={
        <>
          <span className="font-mono text-xs">{automation.id}</span>
          <AutomationKindBadge kind={automation.kind} />
          <AutomationConfigStatusBadge automation={automation} />
        </>
      }
      actions={
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          {automation.enabled ? "Enabled" : "Disabled"}
          <Switch
            aria-label={automation.enabled ? "Disable automation" : "Enable automation"}
            checked={automation.enabled}
            disabled={setEnabled.isPending}
            onCheckedChange={(checked) => setEnabled.mutate({ id: automation.id, enabled: checked })}
          />
        </label>
      }
    />
  );
};
