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
  // The workspace whose canvas the store holds, so a switch never shows or saves the previous one's.
  workspaceId: string;
  nodes: TopologyNode[];
  edges: RelationEdge[];
  viewport: Viewport | null;
  // The automatic layout the stored positions were arranged by (LANE_LAYOUT in TopologyLayout); 0 when never.
  layout: number;
  selectedNodeId: string | null;
  setCanvas: (canvas: Canvas, workspaceId: string) => void;
  setViewport: (viewport: Viewport) => void;
  setLayout: (layout: number) => void;
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
  workspaceId: "",
  nodes: [],
  edges: [],
  viewport: null,
  layout: 0,
  selectedNodeId: null,

  setCanvas: (canvas, workspaceId) =>
    set({
      workspaceId,
      viewport: canvas.viewport ?? null,
      layout: canvas.layout ?? 0,
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
    const { nodes, edges, viewport, layout } = get();
    return {
      schema_version: 2,
      ...(viewport && { viewport }),
      ...(layout > 0 && { layout }),
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
  setLayout: (layout) => set({ layout }),

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
