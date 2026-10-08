import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { TopologyCanvas } from "@/components/topology/TopologyCanvas";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Keyed by workspace, so switching starts a fresh canvas: its own layout pass and camera.
export const TopologyPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const workspaceCrumb = useWorkspaceCrumb();
  return (
    <Container className="flex h-screen flex-col py-8">
      <PageHeader
        crumbs={[workspaceCrumb]}
        title="Topology"
        meta="Every service, the network it runs on, and how traffic reaches it."
      />
      <div className="mt-6 min-h-96 flex-1 overflow-hidden rounded-lg border border-border bg-card shadow-card">
        <TopologyCanvas key={workspaceId} />
      </div>
    </Container>
  );
};
