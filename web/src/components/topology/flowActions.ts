import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  type Connection,
  type EdgeChange,
  type NodeChange,
} from "@xyflow/react";

import { RelationKind, type RelationEdge, type TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

// Split out of flowStore so the always-mounted useLiveEvents doesn't drag xyflow into the eager bundle.
export const onNodesChange = (changes: NodeChange<TopologyNode>[]) => {
  const { nodes, setNodes } = useFlowStore.getState();
  setNodes(applyNodeChanges(changes, nodes));
};

export const onEdgesChange = (changes: EdgeChange<RelationEdge>[]) => {
  const { edges, setEdges } = useFlowStore.getState();
  setEdges(applyEdgeChanges(changes, edges));
};

export const onConnect = (connection: Connection) => {
  const { edges, setEdges } = useFlowStore.getState();
  setEdges(
    addEdge(
      {
        id: `${connection.source}-${connection.sourceHandle ?? ""}-${connection.target}`,
        source: connection.source,
        target: connection.target,
        type: "relation",
        data: { kind: RelationKind.DependsOn },
      } satisfies RelationEdge,
      edges,
    ),
  );
};
