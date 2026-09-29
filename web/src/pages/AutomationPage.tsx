import { useNavigate, useParams } from "react-router";

import { AutomationConfigForm } from "@/components/automation/AutomationConfigForm";
import { AutomationDeleteButton } from "@/components/automation/AutomationDeleteButton";
import { AutomationDetailHeader } from "@/components/automation/AutomationDetailHeader";
import { AutomationHostPicker } from "@/components/automation/AutomationHostPicker";
import { AutomationPendingVersionBanner } from "@/components/automation/AutomationPendingVersionBanner";
import { AutomationRunsFeed } from "@/components/automation/AutomationRunsFeed";
import { AutomationSubscriptionsSection } from "@/components/automation/AutomationSubscriptionsSection";
import { AutomationTokenSection } from "@/components/automation/AutomationTokenSection";
import { AutomationVersionsFeed } from "@/components/automation/AutomationVersionsFeed";
import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { useFetchAutomation } from "@/hooks/AutomationHooks";
import { useFetchAutomationRuns } from "@/hooks/AutomationRunHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

export const AutomationPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const canUpdate = useHasPermission("automations:write");
  const canDelete = useHasPermission("automations:delete");

  const { data: automation, error, isPending } = useFetchAutomation(id);
  const runs = useFetchAutomationRuns(id);

  return (
    <Container className="space-y-6 py-6">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} />}
      {automation && (
        <div className="space-y-6">
          <AutomationDetailHeader automation={automation} />
          <PageTabs
            label="Automation sections"
            tabs={[
              { value: "runs", label: "Runs" },
              { value: "configuration", label: "Configuration" },
              { value: "token", label: "Token" },
              { value: "versions", label: "Versions" },
              { value: "danger", label: "Danger zone", hidden: !canDelete },
            ]}
          >
            <PageTabsContent value="runs">
              {runs.isPending && <LoadingDisplay />}
              {runs.error && <ErrorDisplay error={runs.error} />}
              {runs.data && <AutomationRunsFeed automationId={automation.id} runs={runs.data} />}
            </PageTabsContent>
            <PageTabsContent value="configuration">
              <AutomationPendingVersionBanner automationId={automation.id} />
              <AutomationSubscriptionsSection subscriptions={automation.subscriptions} />
              <AutomationConfigForm automation={automation} />
              {canUpdate && <AutomationHostPicker automation={automation} />}
            </PageTabsContent>
            <PageTabsContent value="token">
              <AutomationTokenSection automation={automation} />
            </PageTabsContent>
            <PageTabsContent value="versions">
              <AutomationVersionsFeed automationId={automation.id} canUpdate={canUpdate} />
            </PageTabsContent>
            <PageTabsContent value="danger">
              <AutomationDeleteButton automationId={automation.id} onDeleted={() => navigate("/automations")} />
            </PageTabsContent>
          </PageTabs>
        </div>
      )}
    </Container>
  );
};
