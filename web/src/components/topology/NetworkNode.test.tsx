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

describe("NetworkNode", () => {
  it("names the network, marks it as a network, and carries the selection", () => {
    const { container, rerender } = render(
      <ReactFlowProvider>
        <NetworkNode {...makeProps()} />
      </ReactFlowProvider>,
    );
    expect(screen.getByText("app-net")).toBeInTheDocument();
    expect(screen.getByText("network")).toBeInTheDocument();
    expect(container.firstElementChild).toHaveAttribute("data-selected", "false");

    rerender(
      <ReactFlowProvider>
        <NetworkNode {...makeProps()} selected />
      </ReactFlowProvider>,
    );
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
