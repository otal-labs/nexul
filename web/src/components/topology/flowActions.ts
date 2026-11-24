import { addEdge, type Connection, type EdgeChangeset, type NodeChangeset } from "@xyflow/react";

import { RelationKind, type RelationEdge, type TopologyNode } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

// Split out of flowStore so the always-mounted useLiveEvents doesn't drag xyflow into the eager bundle.
export const onNodesChange = (changes: NodeChangeset<TopologyNode>) => {
  const { nodes, setNodes } = useFlowStore.getState();
  setNodes(changes.applyTo(nodes));
};

export const onEdgesChange = (changes: EdgeChangeset<RelationEdge>) => {
  const { edges, setEdges } = useFlowStore.getState();
  setEdges(changes.applyTo(edges));
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
