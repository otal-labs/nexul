import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DeployDaySection } from "@/components/service/DeployDaySection";
import { groupDeploysByDay } from "@/components/service/DeployTime";
import { SettingsCard } from "@/components/settings/SettingsCard";
import type { Deploy } from "@/models/Stack";

interface DeployHistorySectionProps {
  deploys: Deploy[] | undefined;
  isLoading: boolean;
}

// Each DeployRow draws its own dot, so a live update never triggers a full-row re-entrance.
export const DeployHistorySection = ({ deploys, isLoading }: DeployHistorySectionProps) => (
  <SettingsCard
    id="deploy-history"
    title="Deploy history"
    description="Every image or build this stack has run, newest first."
  >
    {isLoading && <LoadingDisplay label="Loading deploy history" />}
    {!isLoading && (!deploys || deploys.length === 0) && <EmptyRow>No deploys yet.</EmptyRow>}
    {deploys && deploys.length > 0 && (
      <div className="space-y-6">
        {groupDeploysByDay(deploys).map((group) => (
          <DeployDaySection
            key={`${group.label}-${group.deploys[0]?.id}`}
            label={group.label}
            deploys={group.deploys}
          />
        ))}
      </div>
    )}
  </SettingsCard>
);
