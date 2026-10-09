import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DeployDaySection } from "@/components/service/DeployDaySection";
import { groupDeploysByDay } from "@/components/service/DeployTime";
import { SettingsCard } from "@/components/settings/SettingsCard";
import type { Deploy } from "@/models/Stack";

interface DeployHistorySectionProps {
  deploys: Deploy[] | undefined;
  isLoading: boolean;
  error?: Error | null;
}

// Each DeployRow draws its own status, so a live update never triggers a full-row re-entrance.
export const DeployHistorySection = ({ deploys, isLoading, error = null }: DeployHistorySectionProps) => (
  <SettingsCard
    id="deploy-history"
    title="Deploy history"
    description="Every image or build this stack has run, newest first."
  >
    {isLoading && <LoadingDisplay label="Loading deploy history" />}
    {error && <ErrorDisplay error={error} />}
    {!isLoading && !error && (!deploys || deploys.length === 0) && <EmptyRow flush>No deploys yet. Deploy the stack and each run shows here.</EmptyRow>}
    {deploys && deploys.length > 0 && (
      <ul className="divide-y divide-border overflow-hidden rounded-lg border border-border">
        {groupDeploysByDay(deploys).map((group) => (
          <DeployDaySection
            key={`${group.label}-${group.deploys[0]?.id}`}
            label={group.label}
            deploys={group.deploys}
          />
        ))}
      </ul>
    )}
  </SettingsCard>
);
