import { AutomationsFeed } from "@/components/automation/AutomationsFeed";
import { NewAutomationPanel } from "@/components/automation/NewAutomationPanel";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { useFetchAutomations } from "@/hooks/AutomationHooks";

export const AutomationsPage = () => {
  const { data, error, isPending } = useFetchAutomations();

  return (
    <Container className="space-y-6 py-6">
      <PageHeader
        title="Automations"
        subtitle="First-party code that reacts to what happens in this workspace — Default automations ship with the instance, Custom ones are yours."
      />
      <NewAutomationPanel />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <AutomationsFeed automations={data} />}
    </Container>
  );
};
