import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";

import type { RelationEdge, TopologyNode } from "@/models/Topology";

const NODE_WIDTH = 256;
const NODE_HEIGHT = 96;

const elk = new ELK({
  defaultLayoutOptions: {
    "elk.algorithm": "layered",
    "elk.direction": "DOWN",
    "elk.spacing.nodeNode": "48",
    "elk.layered.spacing.nodeNodeBetweenLayers": "64",
  },
});

export interface LayoutResult {
  nodes: TopologyNode[];
  edges: RelationEdge[];
}

export const layoutGraph = async (nodes: TopologyNode[], edges: RelationEdge[]): Promise<LayoutResult> => {
  const graph: ElkNode = {
    id: "root",
    children: nodes.map((n) => ({ id: n.id, width: NODE_WIDTH, height: NODE_HEIGHT })),
    edges: edges.map((e) => ({ id: e.id, sources: [e.source], targets: [e.target] })),
  };
  const laidOut = await elk.layout(graph);
  const byId = new Map((laidOut.children ?? []).map((c) => [c.id, c]));
  return {
    nodes: nodes.map((n) => {
      const placed = byId.get(n.id);
      if (!placed || placed.x == null || placed.y == null) return n;
      return { ...n, position: { x: placed.x, y: placed.y } };
    }),
    edges,
  };
};
