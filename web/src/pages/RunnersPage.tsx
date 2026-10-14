import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { RunnersPanel } from "@/components/runner/RunnersPanel";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

export const RunnersPage = () => {
  const workspaceCrumb = useWorkspaceCrumb();
  const canAddMachines = useHasInstancePermission("machines:create");
  return (
  <Container className="space-y-6 py-8">
    <PageHeader
      crumbs={[workspaceCrumb]}
      title="Runners"
      meta="Services on your machines that take builds and deploys off the queue."
      actions={canAddMachines && <AddRunnerDialog />}
    />
    <RunnersPanel />
  </Container>
  );
};
