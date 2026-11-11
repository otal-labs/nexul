import { AutomationSdkSetup } from "@/components/automation/AutomationSdkSetup";
import { AutomationSubscriptionsSection } from "@/components/automation/AutomationSubscriptionsSection";
import { AutomationVersionCode } from "@/components/automation/AutomationVersionCode";
import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AutomationKind } from "@/enums/Automation";
import { useFetchAutomationVersionDiff } from "@/hooks/AutomationVersionHooks";
import type { Automation } from "@/models/Automation";

interface AutomationVersionsFeedProps {
  automation: Automation;
}

// The pending diff, how to change the code, and the code that runs now; older versions are in the History dialog.
export const AutomationVersionsFeed = ({ automation }: AutomationVersionsFeedProps) => {
  const { data: diff, error, isPending } = useFetchAutomationVersionDiff(automation.id);
  const custom = automation.kind === AutomationKind.Custom;

  return (
    <div className="space-y-6">
      <AutomationSubscriptionsSection subscriptions={automation.subscriptions} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {diff?.pending && <AutomationVersionDiff automationId={automation.id} pending={diff.pending} active={diff.active} />}
      {diff && custom && <AutomationSdkSetup automationId={automation.id} versions={diff} />}
      {diff?.active && (
        <SettingsCard id="active-version" title={`#${diff.active.sequence} active`} description={diff.active.source ? diff.active.message : "Built bundle"}>
          <AutomationVersionCode automationId={automation.id} sequence={diff.active.sequence} />
        </SettingsCard>
      )}
      {diff && !custom && !diff.active && !diff.pending && <EmptyRow className="border-y border-border">Nothing pushed yet.</EmptyRow>}
    </div>
  );
};
