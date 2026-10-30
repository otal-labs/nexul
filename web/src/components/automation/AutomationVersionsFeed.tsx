import { AutomationSubscriptionsSection } from "@/components/automation/AutomationSubscriptionsSection";
import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchAutomationVersionDiff } from "@/hooks/AutomationVersionHooks";
import type { Automation } from "@/models/Automation";

interface AutomationVersionsFeedProps {
  automation: Automation;
}

// The pending-vs-active diff is where a developer catches what they missed; every older version is in the History dialog.
export const AutomationVersionsFeed = ({ automation }: AutomationVersionsFeedProps) => {
  const diff = useFetchAutomationVersionDiff(automation.id);

  return (
    <div className="space-y-6">
      <AutomationSubscriptionsSection subscriptions={automation.subscriptions} />
      {diff.isPending && <LoadingDisplay />}
      {diff.error && <ErrorDisplay error={diff.error} />}
      {diff.data && <AutomationVersionDiff automationId={automation.id} diff={diff.data} />}
      <p className="text-sm text-muted-foreground">Older versions and Roll back are in History, at the top of the page.</p>
    </div>
  );
};
