import { useNavigate, useParams } from "react-router";

import { AutomationConfigForm } from "@/components/automation/AutomationConfigForm";
import { AutomationDeleteButton } from "@/components/automation/AutomationDeleteButton";
import { AutomationDetailHeader } from "@/components/automation/AutomationDetailHeader";
import { AutomationHostPicker } from "@/components/automation/AutomationHostPicker";
import { AutomationRunsFeed } from "@/components/automation/AutomationRunsFeed";
import { AutomationTokenSection } from "@/components/automation/AutomationTokenSection";
import { AutomationVersionsFeed } from "@/components/automation/AutomationVersionsFeed";
import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { useFetchAutomation } from "@/hooks/AutomationHooks";
import { useActionState } from "@/hooks/useActionState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

export const AutomationPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const canDelete = useActionState({ permission: "automations:delete" }).state !== "hidden";

  const { data: automation, error, isPending } = useFetchAutomation(id);

  return (
    <Container size="page" className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} />}
      {automation && (
        <div className="space-y-6">
          <AutomationDetailHeader automation={automation} />
          <PageTabs
            label="Automation sections"
            tabs={[
              { value: "runs", label: "Runs" },
              { value: "versions", label: "Versions" },
              { value: "settings", label: "Settings" },
            ]}
          >
            <PageTabsContent value="runs">
              <AutomationRunsFeed automation={automation} />
            </PageTabsContent>
            <PageTabsContent value="versions">
              <AutomationVersionsFeed automation={automation} />
            </PageTabsContent>
            <PageTabsContent value="settings">
              <AutomationConfigForm automation={automation} />
              <AutomationHostPicker automation={automation} />
              <AutomationTokenSection automation={automation} />
              {canDelete && (
                <AutomationDeleteButton automationId={automation.id} onDeleted={() => navigate(wsPath("/automations"))} />
              )}
            </PageTabsContent>
          </PageTabs>
        </div>
      )}
    </Container>
  );
};
