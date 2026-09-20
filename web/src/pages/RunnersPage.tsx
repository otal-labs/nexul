import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { RunnersPanel } from "@/components/runner/RunnersPanel";

export const RunnersPage = () => (
  <Container className="space-y-6 py-6">
    <PageHeader
      title="Runners"
      subtitle="The machines that pick up builds and deploys the moment they land in the queue."
      actions={<AddRunnerDialog />}
    />
    <RunnersPanel />
  </Container>
);
