import { useNavigate, useParams, useSearchParams } from "react-router";

import { AutomationConfigForm } from "@/components/automation/AutomationConfigForm";
import { AutomationDeleteButton } from "@/components/automation/AutomationDeleteButton";
import { AutomationDetailHeader } from "@/components/automation/AutomationDetailHeader";
import { AutomationPendingVersionBanner } from "@/components/automation/AutomationPendingVersionBanner";
import { AutomationRunsFeed } from "@/components/automation/AutomationRunsFeed";
import { AutomationSubscriptionsSection } from "@/components/automation/AutomationSubscriptionsSection";
import { AutomationTabsNav, type AutomationTab } from "@/components/automation/AutomationTabsNav";
import { AutomationTokenSection } from "@/components/automation/AutomationTokenSection";
import { AutomationVersionsFeed } from "@/components/automation/AutomationVersionsFeed";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchAutomation } from "@/hooks/AutomationHooks";
import { useFetchAutomationRuns } from "@/hooks/AutomationRunHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

const isTab = (value: string | null): value is AutomationTab => value === "overview" || value === "versions";

export const AutomationPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab: AutomationTab = isTab(searchParams.get("tab")) ? (searchParams.get("tab") as AutomationTab) : "overview";
  const canUpdate = useHasPermission("automations:write");
  const canDelete = useHasPermission("automations:delete");

  const { data: automation, error, isPending } = useFetchAutomation(id);
  const runs = useFetchAutomationRuns(id);

  return (
    <Container className="space-y-6 py-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {automation && (
        <div className="space-y-6">
          <AutomationDetailHeader automation={automation} />
          <AutomationTabsNav active={tab} onSelect={(next) => setSearchParams({ tab: next })} />

          {tab === "overview" && (
            <div className="space-y-4">
              <AutomationPendingVersionBanner automationId={automation.id} />
              <AutomationSubscriptionsSection subscriptions={automation.subscriptions} />
              <AutomationConfigForm automation={automation} />
              <AutomationTokenSection automation={automation} />
              {runs.isPending && <LoadingDisplay />}
              {runs.error && <ErrorDisplay error={runs.error} />}
              {runs.data && <AutomationRunsFeed automationId={automation.id} runs={runs.data} />}
              {canDelete && (
                <AutomationDeleteButton automationId={automation.id} onDeleted={() => navigate("/automations")} />
              )}
            </div>
          )}
          {tab === "versions" && <AutomationVersionsFeed automationId={automation.id} canUpdate={canUpdate} />}
        </div>
      )}
    </Container>
  );
};
