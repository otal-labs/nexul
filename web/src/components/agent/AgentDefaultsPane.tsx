import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { HarnessReadinessLine } from "@/components/settings/HarnessReadinessLine";
import { PairingDefaultsForm } from "@/components/settings/PairingDefaultsForm";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AGENT_PATH } from "@/models/Agent";

// What @Agent and every project's runs use until a project changes it, and where a chat outside a project starts now.
export const AgentDefaultsPane = () => {
  const wsPath = useWorkspacePath();
  const { data: defaults, error, isPending } = useFetchPairingDefaults();

  return (
    <Container className="space-y-6 py-8 sm:pr-3.5">
      <PageHeader crumbs={[{ label: "Agent", to: wsPath(AGENT_PATH) }, { label: "Projects" }]} title="Defaults" meta={<HarnessReadinessLine />} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {defaults && (
        <div className="@container max-w-2xl">
          <PairingDefaultsForm defaults={defaults} />
        </div>
      )}
    </Container>
  );
};
