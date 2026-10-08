import { AutomationSecretsSection } from "@/components/automation/AutomationSecretsSection";
import { AutomationsFeed } from "@/components/automation/AutomationsFeed";
import { NewAutomationDialog } from "@/components/automation/NewAutomationDialog";
import { AutomationHostsSection } from "@/components/automationHost/AutomationHostsSection";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { useFetchAutomations } from "@/hooks/AutomationHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

export const AutomationsPage = () => {
  const { data, error, isPending } = useFetchAutomations();
  const workspaceCrumb = useWorkspaceCrumb();

  return (
    <Container className="space-y-6 py-8">
      <PageHeader
        crumbs={[workspaceCrumb]}
        title="Automations"
        meta="First-party code that reacts to what happens in this workspace — Default automations ship with the instance, Custom ones are yours."
        actions={<NewAutomationDialog />}
      />
      <PageTabs
        label="Automations sections"
        tabs={[
          { value: "automations", label: "Automations" },
          { value: "hosts", label: "Hosts" },
          { value: "secrets", label: "Secrets" },
        ]}
      >
        <PageTabsContent value="automations">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          {data && <AutomationsFeed automations={data} />}
        </PageTabsContent>
        <PageTabsContent value="hosts">
          <AutomationHostsSection />
        </PageTabsContent>
        <PageTabsContent value="secrets">
          <AutomationSecretsSection />
        </PageTabsContent>
      </PageTabs>
    </Container>
  );
};
