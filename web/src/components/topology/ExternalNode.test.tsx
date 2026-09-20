import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { ExternalNode } from "@/components/topology/ExternalNode";
import { ExternalLabel, type ExternalNode as ExternalNodeType } from "@/models/Topology";

type Props = NodeProps<ExternalNodeType>;

const makeProps = (overrides: Partial<Props> = {}): Props => ({
  id: "ext-cf",
  data: { name: "Cloudflare", label: ExternalLabel.Tunnel, url: "https://example.com" },
  selected: false,
  type: "external",
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
      <ExternalNode {...makeProps(overrides)} selected={selected} />
    </ReactFlowProvider>,
  );

describe("ExternalNode", () => {
  it("renders the name, label, and url", () => {
    renderNode();
    expect(screen.getByText("Cloudflare")).toBeInTheDocument();
    expect(screen.getByText(ExternalLabel.Tunnel)).toBeInTheDocument();
    expect(screen.getByText("https://example.com")).toBeInTheDocument();
  });

  it("omits the url when absent", () => {
    renderNode({ data: { name: "Cloudflare", label: ExternalLabel.Tunnel } });
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("applies the selection ring when selected", () => {
    const { container } = renderNode({}, true);
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
