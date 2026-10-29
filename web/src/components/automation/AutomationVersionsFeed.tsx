import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchAutomationVersionDiff } from "@/hooks/AutomationVersionHooks";

interface AutomationVersionsFeedProps {
  automationId: string;
}

// The pending-vs-active diff is where a developer catches what they missed; every older version is in the History dialog.
export const AutomationVersionsFeed = ({ automationId }: AutomationVersionsFeedProps) => {
  const diff = useFetchAutomationVersionDiff(automationId);

  return (
    <div className="space-y-6">
      {diff.isPending && <LoadingDisplay />}
      {diff.error && <ErrorDisplay error={diff.error} />}
      {diff.data && <AutomationVersionDiff automationId={automationId} diff={diff.data} />}
    </div>
  );
};
