import { create } from "zustand";

import {
  RelationKind,
  type Canvas,
  type RelationEdge,
  type TopologyNode,
  type Viewport,
} from "@/models/Topology";

// No @xyflow/react import here, on purpose: it would land xyflow's runtime in the eager bundle.
export type FlowStore = {
  nodes: TopologyNode[];
  edges: RelationEdge[];
  viewport: Viewport | null;
  selectedNodeId: string | null;
  setCanvas: (canvas: Canvas) => void;
  setViewport: (viewport: Viewport) => void;
  toCanvas: () => Canvas;
  setNodes: (nodes: TopologyNode[]) => void;
  setEdges: (edges: RelationEdge[]) => void;
  applyServerPatch: (patch: Partial<Canvas>) => void;
  selectNode: (id: string | null) => void;
};

const toFlowNode = ({ id, type, position, data }: Canvas["nodes"][number]): TopologyNode => ({
  id,
  type,
  position,
  data,
}) as TopologyNode;

const toCanvasNode = ({ id, type, position, data }: TopologyNode): Canvas["nodes"][number] => ({
  id,
  type,
  position,
  data,
}) as Canvas["nodes"][number];

export const useFlowStore = create<FlowStore>((set, get) => ({
  nodes: [],
  edges: [],
  viewport: null,
  selectedNodeId: null,

  setCanvas: (canvas) =>
    set({
      viewport: canvas.viewport ?? null,
      nodes: (canvas.nodes ?? []).map(toFlowNode),
      edges: (canvas.edges ?? []).map(({ id, source, target, data }) => ({
        id,
        source,
        target,
        type: "relation",
        data: { kind: data?.kind ?? RelationKind.DependsOn },
      })),
      selectedNodeId: null,
    }),

  toCanvas: (): Canvas => {
    const { nodes, edges, viewport } = get();
    return {
      schema_version: 2,
      ...(viewport && { viewport }),
      nodes: nodes.map(toCanvasNode),
      edges: edges.map(({ id, source, target, data }) => ({
        id,
        source,
        target,
        type: "relation",
        data: { kind: data?.kind ?? RelationKind.DependsOn },
      })),
    };
  },

  setNodes: (nodes) => set({ nodes }),
  setEdges: (edges) => set({ edges }),
  setViewport: (viewport) => set({ viewport }),

  applyServerPatch: (patch) =>
    set((s) => ({
      nodes: patch.nodes ? patch.nodes.map(toFlowNode) : s.nodes,
      edges: patch.edges
        ? patch.edges.map(({ id, source, target, data }) => ({
            id,
            source,
            target,
            type: "relation",
            data: { kind: data?.kind ?? RelationKind.DependsOn },
          }))
        : s.edges,
    })),

  selectNode: (id) => set({ selectedNodeId: id }),
}));
