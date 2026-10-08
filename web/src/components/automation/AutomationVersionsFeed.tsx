import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionRow } from "@/components/automation/AutomationVersionRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useFetchAutomationVersionDiff, useFetchAutomationVersions } from "@/hooks/AutomationVersionHooks";

interface AutomationVersionsFeedProps {
  automationId: string;
  canUpdate: boolean;
}

// The pending-vs-active diff sits above the full history: it's where a developer catches what they missed.
export const AutomationVersionsFeed = ({ automationId, canUpdate }: AutomationVersionsFeedProps) => {
  const diff = useFetchAutomationVersionDiff(automationId);
  const versions = useFetchAutomationVersions(automationId);

  return (
    <div className="space-y-6">
      {(diff.isPending || versions.isPending) && <LoadingDisplay />}
      {(diff.error || versions.error) && <ErrorDisplay error={diff.error ?? versions.error} />}
      {diff.data && (
        <AutomationVersionDiff automationId={automationId} diff={diff.data} canUpdate={canUpdate} />
      )}
      {versions.data && (
        <section className="space-y-3">
          <h2 className="text-sm font-semibold">Version history</h2>
          {versions.data.length === 0 && <NoDataDisplay message="No versions yet" size="compact" />}
          {versions.data.length > 0 && (
            <ul className="divide-y divide-border overflow-hidden rounded-md border">
              {versions.data.map((version) => (
                <AutomationVersionRow
                  key={version.id}
                  automationId={automationId}
                  version={version}
                  canUpdate={canUpdate}
                />
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
};
