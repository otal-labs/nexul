import { Container } from "@/components/Container";
import { TopologyCanvas } from "@/components/topology/TopologyCanvas";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The canvas is the page: its bar carries the title, so there is no page header above it. Keyed by workspace, so
// switching starts a fresh canvas: its own layout pass and camera.
export const TopologyPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return (
    <Container className="flex h-full flex-col py-3" data-canvas-page>
      <div className="-mx-2 min-h-96 flex-1 overflow-hidden rounded-lg bg-surface-2">
        <TopologyCanvas key={workspaceId} />
      </div>
    </Container>
  );
};
