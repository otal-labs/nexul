import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";

import { useFetchTopology } from "@/hooks/TopologyHooks";
import { useFlowStore } from "@/stores/flowStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const canvas = {
  schema_version: 2,
  nodes: [
    {
      id: "svc-api",
      type: "service",
      position: { x: 0, y: 0 },
      data: { service_id: "svc-api", name: "API", runtime: "docker", status: "healthy" },
    },
  ],
  edges: [],
};

const Harness = () => {
  const query = useFetchTopology();
  const nodeCount = useFlowStore((s) => s.nodes.length);
  return (
    <div>
      <span data-testid="status">{query.isPending ? "loading" : query.isSuccess ? "loaded" : "error"}</span>
      <span data-testid="nodes">{nodeCount}</span>
    </div>
  );
};

describe("useFetchTopology", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useFlowStore.setState({ nodes: [], edges: [], selectedNodeId: null });
  });

  it("fetches the canvas and applies it to the flow store", async () => {
    mocks.get.mockResolvedValue({ data: canvas });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <Harness />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(mocks.get).toHaveBeenCalledWith("/api/topology", { params: { environment: "default" } });
    await screen.findByText("loaded");
    await vi.waitFor(() => {
      expect(screen.getByTestId("nodes").textContent).toBe("1");
    });
  });

  it("surfaces errors without crashing", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <MemoryRouter>
          <Harness />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByText("error");
  });
});
