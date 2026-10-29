import { render, screen } from "@testing-library/react";
import { Position, ReactFlowProvider, type EdgeProps } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { RelationEdge } from "@/components/topology/RelationEdge";

// The label renders through React Flow's label layer, which only exists inside a mounted canvas.
vi.mock("@xyflow/react", async (original) => ({
  ...(await original<typeof import("@xyflow/react")>()),
  EdgeLabelRenderer: ({ children }: { children: React.ReactNode }) => <foreignObject>{children}</foreignObject>,
}));

const makeProps = (overrides: Partial<EdgeProps> = {}): EdgeProps => ({
  id: "edge-1",
  source: "svc-api",
  target: "svc-db",
  sourceX: 0,
  sourceY: 0,
  targetX: 100,
  targetY: 100,
  sourcePosition: Position.Right,
  targetPosition: Position.Left,
  selected: false,
  ...overrides,
} as EdgeProps);

const renderEdge = (overrides: Partial<EdgeProps> = {}) =>
  render(
    <svg>
      <ReactFlowProvider>
        <RelationEdge {...makeProps(overrides)} />
      </ReactFlowProvider>
    </svg>,
  );

describe("RelationEdge", () => {
  it.each([
    ["depends_on", "depends on"],
    ["mounts", "mounts"],
  ])("labels %s edges", (kind, label) => {
    renderEdge({ data: { kind } });
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("rings the label in the selection colour when the edge is selected", () => {
    renderEdge({ data: { kind: "connects_to" }, selected: true });
    expect(screen.getByText("connects to")).toHaveClass("outline-ring", "text-primary");
  });

  it.each([
    ["depends_on", "4 4"],
    ["connects_to", "1 4"],
  ])("strokes %s edges in their own dash", (kind, dash) => {
    const { container } = renderEdge({ data: { kind } });
    expect(container.querySelector("path.react-flow__edge-path")).toHaveStyle({ strokeDasharray: dash });
  });

  it("prefers an explicit label over the kind's", () => {
    renderEdge({ data: { kind: "connects_to", label: ":3000" } });
    expect(screen.getByText(":3000")).toBeInTheDocument();
    expect(screen.queryByText("connects to")).not.toBeInTheDocument();
  });

  it("renders no label when kind is missing", () => {
    renderEdge();
    expect(screen.queryByText(/depends on|connects to|mounts/)).not.toBeInTheDocument();
  });
});
