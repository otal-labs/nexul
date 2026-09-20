import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import {
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  ReactFlow,
  ViewportPortal,
  type NodeMouseHandler,
  type OnMoveEnd,
  type OnNodeDrag,
} from "@xyflow/react";

import { AddNodeDialog } from "@/components/topology/AddNodeDialog";
import { CanvasEmptyHint } from "@/components/topology/CanvasEmptyHint";
import { ExternalNode as ExternalNodeView } from "@/components/topology/ExternalNode";
import { onConnect, onEdgesChange } from "@/components/topology/flowActions";
import { FreeNodeDialog } from "@/components/topology/FreeNodeDialog";
import { GatewayNode as GatewayNodeView } from "@/components/topology/GatewayNode";
import { HostnameNode as HostnameNodeView } from "@/components/topology/HostnameNode";
import { NetworkBox } from "@/components/topology/NetworkBox";
import { NetworkNode as NetworkNodeView } from "@/components/topology/NetworkNode";
import { RelationEdge as RelationEdgeView } from "@/components/topology/RelationEdge";
import { RouteEdge as RouteEdgeView } from "@/components/topology/RouteEdge";
import { ServiceNode as ServiceNodeView } from "@/components/topology/ServiceNode";
import { TopologyPalette } from "@/components/topology/TopologyPalette";
import { useFetchTopology, useSaveTopology } from "@/hooks/TopologyHooks";
import { useTopologyView } from "@/hooks/useTopologyView";
import { NodeType, type ExternalNode, type NetworkNode, type RelationEdge, type TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const nodeTypes = {
  service: ServiceNodeView,
  network: NetworkNodeView,
  external: ExternalNodeView,
  hostname: HostnameNodeView,
  gateway: GatewayNodeView,
};
const edgeTypes = { relation: RelationEdgeView, route: RouteEdgeView };

// Needs the ReactFlowProvider mounted by TopologyCanvas.
export const TopologyFlow = () => {
  const navigate = useNavigate();
  const nodes = useFlowStore((s) => s.nodes);
  const storeViewport = useFlowStore((s) => s.setViewport);
  const selectedNodeId = useFlowStore((s) => s.selectedNodeId);
  const selectNode = useFlowStore((s) => s.selectNode);
  const save = useSaveTopology();
  const { isPending } = useFetchTopology();
  const { viewNodes, viewEdges, networkRects, handleNodesChange } = useTopologyView();

  const [addingExternal, setAddingExternal] = useState(false);

  // Pan and zoom are part of the map: saved a moment after the camera stops, so a reload opens where it was left.
  const viewportSave = useRef<number | undefined>(undefined);
  const handleMoveEnd: OnMoveEnd = (_, viewport) => {
    if (isPending) return;
    // Restoring the saved camera on load ends a move too; the same values are not worth a write.
    const saved = useFlowStore.getState().viewport;
    if (saved && saved.x === viewport.x && saved.y === viewport.y && saved.zoom === viewport.zoom) return;
    storeViewport(viewport);
    window.clearTimeout(viewportSave.current);
    viewportSave.current = window.setTimeout(() => void save.mutateAsync(), 600);
  };

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) ?? null;
  const freeNode =
    selectedNode?.type === NodeType.Network || selectedNode?.type === NodeType.External
      ? (selectedNode as NetworkNode | ExternalNode)
      : null;

  // A service (and a gateway is one) has a whole page — its stack's page; only hand-drawn nodes edit in place.
  // stackId is wired in at render time from the containers list (wiring.ts); a node deployed before that wiring
  // ran has none yet, so the click is a no-op rather than a broken link.
  const handleNodeClick: NodeMouseHandler<TopologyNode> = (_, node) => {
    if (node.type === NodeType.Service || node.type === "gateway") {
      if (node.data.stackId) void navigate(`/stacks/${node.data.stackId}`);
      return;
    }
    if (node.type === NodeType.Network || node.type === NodeType.External) selectNode(node.id);
  };
  // Pills aren't stored, so moving one has nothing to save.
  const handleNodeDragStop: OnNodeDrag<TopologyNode> = (_, node) => {
    if (node.type === "hostname") return;
    void save.mutateAsync();
  };

  return (
    <div className="relative h-full w-full">
      <ReactFlow<TopologyNode, RelationEdge>
        nodes={viewNodes}
        edges={viewEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={handleNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={handleNodeClick}
        onNodeDragStop={handleNodeDragStop}
        onMoveEnd={handleMoveEnd}
        className="bg-background"
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1.5} />
        <Controls />
        {/* Minimap is overview-only chrome, dropped below `md`; pan/zoom/drag/connect stay live at every width. */}
        <MiniMap pannable zoomable className="hidden md:block" />
        <ViewportPortal>
          {networkRects.map((rect) => (
            <NetworkBox key={rect.name} rect={rect} />
          ))}
        </ViewportPortal>
      </ReactFlow>
      <TopologyPalette onAddExternal={() => setAddingExternal(true)} />
      {!isPending && nodes.length === 0 && <CanvasEmptyHint onAddNode={() => setAddingExternal(true)} />}
      <AddNodeDialog kind={NodeType.External} open={addingExternal} onClose={() => setAddingExternal(false)} />
      <FreeNodeDialog node={freeNode} onClose={() => selectNode(null)} />
    </div>
  );
};
