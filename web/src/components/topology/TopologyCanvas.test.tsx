import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TopologyCanvas } from "@/components/topology/TopologyCanvas";
import { ServiceStatus, type Canvas } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { resetViewer, viewer } from "@/test/viewerActions";

const access = vi.hoisted(() => ({ areas: ["stacks"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

beforeEach(() => {
  access.areas = ["stacks"];
  resetViewer();
  viewer.instance = ["topology:write", "topology:delete"];
});

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put }, errorMessage: () => "" }));

// A service's name shows on its card and in the gateway row that reaches it; the card is the one to click.
const serviceCard = async (name: string) => (await screen.findAllByText(name)).find((el) => el.closest(".react-flow__node-service"))!;

const wrap = (ui: React.ReactNode) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <MemoryRouter initialEntries={["/acme/topology"]}>
      <Routes>
        <Route path="/acme/topology" element={ui} />
        <Route path="/acme/stacks/:stackId" element={<p>stack page</p>} />
        <Route path="/acme/stacks/:stackId/logs/:service" element={<p>logs page</p>} />
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
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    useFlowStore.getState().setCanvas({ schema_version: 2, nodes: [], edges: [] }, "ws-1");
  });

  it("renders the service nodes from the fetched canvas", async () => {
    render(wrap(<TopologyCanvas />));
    expect(await serviceCard("api-gateway")).toBeInTheDocument();
    expect(screen.getByText("cloudflared-local")).toBeInTheDocument();
  });

  it("lays out never-placed nodes in lanes on first paint and saves the result with the layout's version", async () => {
    render(wrap(<TopologyCanvas />));
    await waitFor(() => {
      const nodes = useFlowStore.getState().nodes;
      expect(nodes).toHaveLength(2);
      expect(nodes.some((n) => n.position.x !== 0 || n.position.y !== 0)).toBe(true);
    });
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/topology", expect.objectContaining({ schema_version: 2, layout: 1 }), expect.anything()));
  });

  it("lays a canvas arranged by an older layout out again once, even when every node was placed", async () => {
    const old: Canvas = { ...canvas, nodes: canvas.nodes.map((n, i) => ({ ...n, position: { x: 100 + i * 400, y: 250 } })) };
    mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/topology" ? old : (responses[url] ?? []) }));
    render(wrap(<TopologyCanvas />));
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/topology", expect.objectContaining({ layout: 1 }), expect.anything()));
    expect(useFlowStore.getState().nodes.map((n) => n.position)).not.toEqual(old.nodes.map((n) => n.position));
  });

  it("counts the services and hostnames in the bar, and offers Arrange only to someone who can write", async () => {
    render(wrap(<TopologyCanvas />));
    await serviceCard("api-gateway");
    expect(screen.getByRole("heading", { name: "Topology" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Arrange in lanes" }).length).toBeGreaterThan(0);
  });

  it("turns a count that needs attention into a button that steps through those services", async () => {
    const failing: Canvas = { ...canvas, nodes: canvas.nodes.map((n) => (n.id === "svc-api" ? { ...n, data: { ...n.data, status: ServiceStatus.Failed } } : n)) as Canvas["nodes"] };
    mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/topology" ? failing : (responses[url] ?? []) }));
    render(wrap(<TopologyCanvas />));
    fireEvent.click(await screen.findByRole("button", { name: "Show the next failed service, of 1" }));
    expect(screen.queryByRole("button", { name: /next running service/ })).not.toBeInTheDocument();
  });

  it("lays out for a viewer who can't write topology without saving it, and offers no External node", async () => {
    viewer.instance = [];
    render(wrap(<TopologyCanvas />));
    await waitFor(() => expect(useFlowStore.getState().nodes.some((n) => n.position.x !== 0 || n.position.y !== 0)).toBe(true));

    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Add external node" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Arrange in lanes" })).not.toBeInTheDocument();
  });

  it("locks the External node button for a viewer whose role shows locked actions", async () => {
    viewer.instance = [];
    viewer.showLocked = true;
    render(wrap(<TopologyCanvas />));

    expect(await screen.findByRole("button", { name: "Add external node" })).toHaveAttribute("aria-disabled", "true");
  });

  it("keeps the owner's saved positions and viewport instead of laying out again", async () => {
    const placed: Canvas = {
      ...canvas,
      nodes: canvas.nodes.map((n, i) => ({ ...n, position: { x: 100 + i * 700, y: 250 } })),
      viewport: { x: 40, y: 60, zoom: 0.5 },
      layout: 1,
    };
    mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/topology" ? placed : (responses[url] ?? []) }));
    render(wrap(<TopologyCanvas />));
    await waitFor(() => expect(document.querySelector('a[href="https://api.example.com"]')).toBeInTheDocument());
    expect(useFlowStore.getState().nodes.map((n) => n.position)).toEqual([
      { x: 100, y: 250 },
      { x: 800, y: 250 },
    ]);
    expect(useFlowStore.getState().viewport).toEqual({ x: 40, y: 60, zoom: 0.5 });
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("draws the hostname pill, the gateway rows, and the docker network box from live data", async () => {
    render(wrap(<TopologyCanvas />));
    // React Flow keeps nodes visibility:hidden until measured, which jsdom never does, so role queries can't name it.
    await waitFor(() => expect(document.querySelector('a[href="https://api.example.com"]')).toBeInTheDocument());
    expect(document.querySelector("[data-network=\"app-net\"]")).toHaveTextContent("network · app-net");
    expect(screen.getByText("Cloudflare tunnel")).toBeInTheDocument();
    expect(document.querySelector('[data-route="exp-1"]')).toHaveTextContent("→ api-gateway:3000");
    // Edges only draw once nodes are measured; their wiring is covered by wiring.test.ts.
    expect(screen.getAllByText("instance").length).toBeGreaterThan(0);
  });

  it("opens the stack page when a service node is clicked", async () => {
    render(wrap(<TopologyCanvas />));
    fireEvent.click(await serviceCard("api-gateway"));
    expect(await screen.findByText("stack page")).toBeInTheDocument();
  });

  it("opens the service's logs when the viewer can read them", async () => {
    access.areas = ["stacks", "stackLogs"];
    render(wrap(<TopologyCanvas />));
    fireEvent.click(await serviceCard("api-gateway"));
    expect(await screen.findByText("logs page")).toBeInTheDocument();
  });

  it("does not open the stack page for a viewer who can't read stacks", async () => {
    access.areas = [];
    render(wrap(<TopologyCanvas />));
    fireEvent.click(await serviceCard("api-gateway"));
    expect(screen.queryByText("stack page")).not.toBeInTheDocument();
  });
});
