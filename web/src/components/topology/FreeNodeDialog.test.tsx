import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FreeNodeDialog } from "@/components/topology/FreeNodeDialog";
import {
  NodeType,
  type NetworkNode,
} from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), _delete: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put, delete: mocks._delete } }));

const networkNode: NetworkNode = {
  id: "net-main",
  type: NodeType.Network,
  position: { x: 0, y: 0 },
  data: { name: "app-net" },
};

const wrap = (ui: React.ReactNode) => (
  <QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>
);

describe("FreeNodeDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks._delete.mockReset();
    mocks.get.mockResolvedValue({ data: { schema_version: 2, nodes: [], edges: [] } });
    mocks.put.mockResolvedValue({ data: { schema_version: 2, nodes: [], edges: [] } });
    mocks._delete.mockResolvedValue({ data: { schema_version: 2, nodes: [], edges: [] } });
    useFlowStore.setState({ nodes: [networkNode], edges: [], selectedNodeId: null });
  });

  it("saves an edited network node name via full canvas PUT", async () => {
    const onClose = vi.fn();
    render(wrap(<FreeNodeDialog node={networkNode} onClose={onClose} />));

    fireEvent.change(screen.getByLabelText(/network name/i), { target: { value: "frontend-net" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mocks.put).toHaveBeenCalledWith(
        "/api/topology",
        expect.objectContaining({ schema_version: 2 }),
        { params: { environment: "default" } },
      );
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("deletes a free node via DELETE /api/topology/nodes/{id}", async () => {
    const onClose = vi.fn();
    render(wrap(<FreeNodeDialog node={networkNode} onClose={onClose} />));

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(mocks._delete).toHaveBeenCalledWith("/api/topology/nodes/net-main", {
        params: { environment: "default" },
      });
    });
    expect(onClose).toHaveBeenCalled();
  });
});
