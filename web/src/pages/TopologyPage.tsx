import { TopologyCanvas } from "@/components/topology/TopologyCanvas";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The canvas is the page, edge to edge in the panel: its bar carries the title, so there is no page header above it.
// Keyed by workspace, so switching starts a fresh canvas: its own layout pass and camera.
export const TopologyPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return (
    <div className="flex h-full min-h-96 flex-col overflow-hidden" data-canvas-page>
      <TopologyCanvas key={workspaceId} />
    </div>
  );
};
