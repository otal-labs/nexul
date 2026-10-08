import { EnterList } from "@/components/EnterList";
import { AutomationVersionDiff } from "@/components/automation/AutomationVersionDiff";
import { AutomationVersionRow } from "@/components/automation/AutomationVersionRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
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
        <SettingsCard id="version-history" title="Version history" description="Every version pushed for this automation, newest first.">
          {versions.data.length === 0 && <EmptyRow>No versions yet</EmptyRow>}
          {versions.data.length > 0 && (
            <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
              {versions.data.map((version) => (
                <AutomationVersionRow
                  key={version.id}
                  automationId={automationId}
                  version={version}
                  canUpdate={canUpdate}
                />
              ))}
            </EnterList>
          )}
        </SettingsCard>
      )}
    </div>
  );
};
