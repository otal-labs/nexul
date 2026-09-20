import { beforeEach, describe, expect, it } from "vitest";

import { onConnect } from "@/components/topology/flowActions";
import {
  ServiceStatus,
  type Canvas,
  type RelationEdge,
  type ServiceNode,
} from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const node: ServiceNode = {
  id: "svc-api",
  type: "service",
  position: { x: 0, y: 0 },
  data: {
    service_id: "svc-api",
    name: "api-gateway",
    runtime: "go",
    url: "https://api.example.com",
    status: ServiceStatus.Healthy,
    replicas: 3,
  },
};

const canvas: Canvas = {
  schema_version: 2,
  nodes: [node],
  edges: [
    { id: "edge-1", source: "svc-api", target: "svc-db", type: "relation", data: { kind: "depends_on" } },
  ],
};

describe("useFlowStore", () => {
  beforeEach(() => {
    useFlowStore.setState({ nodes: [], edges: [], viewport: null, selectedNodeId: null });
  });

  it("loads a canvas into nodes and edges", () => {
    useFlowStore.getState().setCanvas(canvas);
    const { nodes, edges } = useFlowStore.getState();
    expect(nodes).toHaveLength(1);
    expect(nodes[0]?.id).toBe("svc-api");
    expect(edges[0]?.data?.kind).toBe("depends_on");
  });

 it("serializes nodes and edges back to the stored canvas shape", () => {
    useFlowStore.getState().setCanvas(canvas);
    const out = useFlowStore.getState().toCanvas();
    expect(out.schema_version).toBe(2);
    expect(out.nodes[0]?.data.name).toBe("api-gateway");
    expect(out.edges[0]).toEqual(canvas.edges[0]);
  });

  it("keeps the viewport with the canvas and only writes it back once set", () => {
    useFlowStore.getState().setCanvas({ schema_version: 2, nodes: [node], edges: [] });
    expect(useFlowStore.getState().viewport).toBeNull();
    expect(useFlowStore.getState().toCanvas()).not.toHaveProperty("viewport");
    useFlowStore.getState().setViewport({ x: 10, y: -20, zoom: 0.75 });
    expect(useFlowStore.getState().toCanvas().viewport).toEqual({ x: 10, y: -20, zoom: 0.75 });
    useFlowStore.getState().setCanvas({ ...canvas, viewport: { x: 1, y: 2, zoom: 1 } });
    expect(useFlowStore.getState().viewport).toEqual({ x: 1, y: 2, zoom: 1 });
  });

  it("adds a relation edge on connect", () => {
    useFlowStore.getState().setCanvas({ schema_version: 2, nodes: [node], edges: [] });
    onConnect({
      source: "svc-api",
      target: "svc-db",
      sourceHandle: null,
      targetHandle: null,
    });
    const edges = useFlowStore.getState().edges as RelationEdge[];
    expect(edges).toHaveLength(1);
    expect(edges[0]?.type).toBe("relation");
    expect(edges[0]?.data?.kind).toBe("depends_on");
  });

  it("selects and clears a node", () => {
    useFlowStore.getState().selectNode("svc-api");
    expect(useFlowStore.getState().selectedNodeId).toBe("svc-api");
    useFlowStore.getState().selectNode(null);
    expect(useFlowStore.getState().selectedNodeId).toBeNull();
  });

  it("tolerates null nodes/edges from the API (empty canvas)", () => {
    useFlowStore.getState().setCanvas({
      schema_version: 2,
      nodes: null as unknown as never[],
      edges: null as unknown as never[],
    });
    const { nodes, edges } = useFlowStore.getState();
    expect(nodes).toHaveLength(0);
    expect(edges).toHaveLength(0);
  });

  it("replaces nodes via setNodes", () => {
    useFlowStore.getState().setCanvas(canvas);
    const moved = { ...node, position: { x: 10, y: 20 } };
    useFlowStore.getState().setNodes([moved]);
    expect(useFlowStore.getState().nodes[0]?.position).toEqual({ x: 10, y: 20 });
  });
});
