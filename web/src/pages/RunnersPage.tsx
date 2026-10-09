import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { RunnersPanel } from "@/components/runner/RunnersPanel";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

export const RunnersPage = () => {
  const workspaceCrumb = useWorkspaceCrumb();
  return (
  <Container className="space-y-6 py-8">
    <PageHeader
      crumbs={[workspaceCrumb]}
      title="Runners"
      meta="Services on your machines that take builds and deploys off the queue."
      actions={<AddRunnerDialog />}
    />
    <RunnersPanel />
  </Container>
  );
};
