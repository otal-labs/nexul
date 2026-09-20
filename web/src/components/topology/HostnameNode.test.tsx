import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { HostnameNode } from "@/components/topology/HostnameNode";
import type { HostnameNode as HostnameNodeType } from "@/models/Topology";

const props: NodeProps<HostnameNodeType> = {
  id: "host-1",
  data: { hostname: "llmtested.example.com", kind: "tunnel" },
  selected: false,
  type: "hostname",
  dragging: false,
  draggable: false,
  selectable: false,
  deletable: false,
  zIndex: 0,
  isConnectable: false,
  positionAbsoluteX: 0,
  positionAbsoluteY: 0,
};

describe("HostnameNode", () => {
  it("links out to the hostname", () => {
    render(
      <ReactFlowProvider>
        <HostnameNode {...props} />
      </ReactFlowProvider>,
    );
    const link = screen.getByRole("link", { name: "llmtested.example.com" });
    expect(link).toHaveAttribute("href", "https://llmtested.example.com");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
