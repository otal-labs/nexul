import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NetworkBox } from "@/components/topology/NetworkBox";
import { ServiceStatus, type TopologyNode } from "@/models/Topology";
import { buildNetworkRects } from "@/utils/TopologyLayout";

const makeService = (id: string, position: { x: number; y: number }): TopologyNode => ({
  id,
  type: "service",
  position,
  data: { service_id: id, name: id, status: ServiceStatus.Healthy },
});

describe("buildNetworkRects", () => {
  it("wraps a network's members with room for the label", () => {
    const rects = buildNetworkRects(
      [{ name: "app-net", memberIds: ["api", "db"] }],
      [makeService("api", { x: 100, y: 100 }), makeService("db", { x: 400, y: 100 })],
    );
    expect(rects).toEqual([{ name: "app-net", x: 76, y: 52, width: 400 + 256 - 100 + 48, height: 112 + 48 + 24 }]);
  });

  it("skips a network none of whose members are on the canvas", () => {
    expect(buildNetworkRects([{ name: "ghost", memberIds: ["missing"] }], [])).toEqual([]);
  });
});

describe("NetworkBox", () => {
  it("renders a dashed, labelled box pinned behind the nodes", () => {
    const { container } = render(<NetworkBox rect={{ name: "app-net", x: 10, y: 20, width: 200, height: 100 }} />);
    const box = container.firstElementChild;
    expect(box).toHaveClass("border-dashed");
    expect(box).toHaveStyle({ zIndex: -1, pointerEvents: "none", left: "10px", top: "20px", width: "200px", height: "100px" });
    expect(screen.getByText("network · app-net")).toBeInTheDocument();
  });
});
