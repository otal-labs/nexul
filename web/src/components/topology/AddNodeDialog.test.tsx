import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddNodeDialog } from "@/components/topology/AddNodeDialog";
import { NodeType } from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";
import { pickOption } from "@/test/pickOption";
import userEvent from "@testing-library/user-event";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post } }));

const wrap = (ui: React.ReactNode) => (
  <QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>
);

describe("AddNodeDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    useFlowStore.setState({ nodes: [], edges: [], selectedNodeId: null });
    mocks.get.mockResolvedValue({ data: { schema_version: 2, nodes: [], edges: [] } });
  });

  it("adds a network node via POST /api/topology/nodes", async () => {
    mocks.post.mockResolvedValue({
      data: {
        schema_version: 2,
        nodes: [{ id: "net-abc", type: "network", position: { x: 0, y: 0 }, data: { name: "app-net" } }],
        edges: [],
      },
    });
    const onClose = vi.fn();
    render(wrap(<AddNodeDialog kind={NodeType.Network} open onClose={onClose} />));

    fireEvent.change(screen.getByLabelText(/network name/i), { target: { value: "app-net" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));

    await waitFor(() => {
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/topology/nodes",
        expect.objectContaining({ type: "network", data: { name: "app-net" } }),
        { params: { environment: "default" } },
      );
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("adds an external node with a label picker", async () => {
    mocks.post.mockResolvedValue({
      data: {
        schema_version: 2,
        nodes: [{ id: "ext-abc", type: "external", position: { x: 0, y: 0 }, data: { name: "Cloudflare", label: "tunnel", url: "https://example.com" } }],
        edges: [],
      },
    });
    const onClose = vi.fn();
    render(wrap(<AddNodeDialog kind={NodeType.External} open onClose={onClose} />));

    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: "Cloudflare" } });
    await pickOption(userEvent.setup(), /label/i, "Tunnel");
    fireEvent.change(screen.getByLabelText(/url/i), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));

    await waitFor(() => {
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/topology/nodes",
        expect.objectContaining({ type: "external", data: { name: "Cloudflare", label: "tunnel", url: "https://example.com" } }),
        { params: { environment: "default" } },
      );
    });
    expect(onClose).toHaveBeenCalled();
  });
});
