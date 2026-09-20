import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { NetworkNode } from "@/components/topology/NetworkNode";
import type { NetworkNode as NetworkNodeType } from "@/models/Topology";

type Props = NodeProps<NetworkNodeType>;

const makeProps = (overrides: Partial<Props> = {}): Props => ({
  id: "net-main",
  data: { name: "app-net" },
  selected: false,
  type: "network",
  dragging: false,
  draggable: true,
  selectable: true,
  deletable: true,
  zIndex: 0,
  isConnectable: true,
  positionAbsoluteX: 0,
  positionAbsoluteY: 0,
  ...overrides,
});

const renderNode = (overrides: Partial<Props> = {}, selected = false) =>
  render(
    <ReactFlowProvider>
      <NetworkNode {...makeProps(overrides)} selected={selected} />
    </ReactFlowProvider>,
  );

describe("NetworkNode", () => {
  it("renders the network name", () => {
    renderNode();
    expect(screen.getByText("app-net")).toBeInTheDocument();
  });

  it("marks the node as a network", () => {
    renderNode();
    expect(screen.getByText("network")).toBeInTheDocument();
  });

  it("applies the selection ring when selected", () => {
    const { container } = renderNode({}, true);
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
