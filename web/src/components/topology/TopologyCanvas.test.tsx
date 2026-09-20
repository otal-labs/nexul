import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TopologyCanvas } from "@/components/topology/TopologyCanvas";
import { ServiceStatus, type Canvas } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put }, errorMessage: () => "" }));

const wrap = (ui: React.ReactNode) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <MemoryRouter initialEntries={["/topology"]}>
      <Routes>
        <Route path="/topology" element={ui} />
        <Route path="/stacks/:stackId" element={<p>stack page</p>} />
      </Routes>
    </MemoryRouter>
  </QueryClientProvider>
);

const canvas: Canvas = {
  schema_version: 2,
  nodes: [
    {
      id: "svc-api",
      type: "service",
      position: { x: 0, y: 0 },
      data: { service_id: "svc-api", name: "api-gateway", runtime: "go", status: ServiceStatus.Healthy },
    },
    {
      id: "svc-cf",
      type: "service",
      position: { x: 0, y: 0 },
      data: { service_id: "svc-cf", name: "cloudflared-local", runtime: "docker", status: ServiceStatus.Running },
    },
  ],
  edges: [],
};

const responses: Record<string, unknown> = {
  "/api/topology": canvas,
  "/api/stacks": [{ id: "stack-1", project_id: "p", name: "api", slug: "api", machine: "instance", strategy: "run", managed: true, created_at: "", updated_at: "" }],
  "/api/stacks/stack-1/services": [
    { id: "svc-api", stack_id: "stack-1", name: "api-gateway", declared: {}, status: "healthy", networks: [{ name: "app-net" }] },
    { id: "svc-cf", stack_id: "stack-1", name: "cloudflared-local", declared: {}, status: "running", networks: [{ name: "app-net" }] },
  ],
  "/api/dns/gateways": [{ id: "gw-1", kind: "tunnel", docker_network: "app-net", service_id: "svc-cf", zone_id: "z", zone: "example.com" }],
  "/api/dns/exposures": [{ id: "exp-1", gateway_id: "gw-1", hostname: "api.example.com", service_id: "svc-api", service: "api-gateway", port: 3000, zone_id: "z", zone: "example.com" }],
};

describe("TopologyCanvas", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.get.mockImplementation((url: string) => Promise.resolve({ data: responses[url] ?? [] }));
    mocks.put.mockImplementation((_url: string, body: Canvas) => Promise.resolve({ data: body }));
    useFlowStore.getState().setCanvas({ schema_version: 2, nodes: [], edges: [] });
  });

  it("renders the service nodes from the fetched canvas", async () => {
    render(wrap(<TopologyCanvas />));
    expect(await screen.findByText("api-gateway")).toBeInTheDocument();
    expect(screen.getByText("cloudflared-local")).toBeInTheDocument();
  });

  it("lays out never-placed nodes with elkjs on first paint and saves the result", async () => {
    render(wrap(<TopologyCanvas />));
    await waitFor(() => {
      const nodes = useFlowStore.getState().nodes;
      expect(nodes).toHaveLength(2);
      expect(nodes.some((n) => n.position.x !== 0 || n.position.y !== 0)).toBe(true);
    });
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/topology", expect.objectContaining({ schema_version: 2 }), expect.anything()));
  });

  it("keeps the owner's saved positions and viewport instead of laying out again", async () => {
    const placed: Canvas = {
      ...canvas,
      nodes: canvas.nodes.map((n, i) => ({ ...n, position: { x: 100 + i * 300, y: 250 } })),
      viewport: { x: 40, y: 60, zoom: 0.5 },
    };
    mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/topology" ? placed : (responses[url] ?? []) }));
    render(wrap(<TopologyCanvas />));
    await waitFor(() => expect(document.querySelector('a[href="https://api.example.com"]')).toBeInTheDocument());
    expect(useFlowStore.getState().nodes.map((n) => n.position)).toEqual([
      { x: 100, y: 250 },
      { x: 400, y: 250 },
    ]);
    expect(useFlowStore.getState().viewport).toEqual({ x: 40, y: 60, zoom: 0.5 });
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("draws the hostname pill, the gateway rows, and the docker network box from live data", async () => {
    render(wrap(<TopologyCanvas />));
    // React Flow keeps nodes visibility:hidden until measured, which jsdom never does, so role queries can't name it.
    await waitFor(() => expect(document.querySelector('a[href="https://api.example.com"]')).toBeInTheDocument());
    expect(screen.getByText("network · app-net")).toBeInTheDocument();
    expect(screen.getByText("Cloudflare tunnel")).toBeInTheDocument();
    expect(screen.getByText("api-gateway:3000")).toBeInTheDocument();
    // Edges only draw once nodes are measured; their wiring is covered by wiring.test.ts.
    expect(screen.getByText("instance · run")).toBeInTheDocument();
  });

  it("opens the stack page when a service node is clicked", async () => {
    render(wrap(<TopologyCanvas />));
    fireEvent.click(await screen.findByText("api-gateway"));
    expect(await screen.findByText("stack page")).toBeInTheDocument();
  });
});
