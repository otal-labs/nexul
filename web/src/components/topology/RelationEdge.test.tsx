import { render, screen } from "@testing-library/react";
import { Position, ReactFlowProvider, type EdgeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { RelationEdge } from "@/components/topology/RelationEdge";

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
  ])("labels %s edges with a pill", (kind, label) => {
    const { container } = renderEdge({ data: { kind } });
    expect(screen.getByText(label)).toBeInTheDocument();
    expect(container.querySelector("rect")).toBeInTheDocument();
  });

  it("turns the pill ember when the edge is selected", () => {
    const { container } = renderEdge({ data: { kind: "connects_to" }, selected: true });
    const pill = container.querySelector("rect");
    const label = container.querySelector("text");
    expect(pill).toHaveClass("stroke-ring");
    expect(label).toHaveClass("fill-primary");
  });

  it("prefers an explicit label over the kind's", () => {
    renderEdge({ data: { kind: "connects_to", label: ":3000" } });
    expect(screen.getByText(":3000")).toBeInTheDocument();
    expect(screen.queryByText("connects to")).not.toBeInTheDocument();
  });

  it("renders no label when kind is missing", () => {
    renderEdge();
    expect(screen.queryByText(/depends on|connects to|mounts/)).not.toBeInTheDocument();
    expect(document.querySelector("rect")).not.toBeInTheDocument();
  });
});
