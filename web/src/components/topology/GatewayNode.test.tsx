import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { GatewayNode } from "@/components/topology/GatewayNode";
import { ServiceStatus, type GatewayNode as GatewayNodeType } from "@/models/Topology";

const props: NodeProps<GatewayNodeType> = {
  id: "svc-cf",
  data: {
    service_id: "svc-cf",
    name: "cloudflared-local",
    status: ServiceStatus.Healthy,
    target: "instance",
    kind: "tunnel",
    networks: ["nexul_default"],
    routes: [
      { id: "e1", hostname: "llmtested.example.com", service: "hello-api", port: 3000, address: "172.18.0.4" },
      { id: "e2", hostname: "example.com", service: "web", port: 80 },
    ],
  },
  selected: false,
  type: "gateway",
  dragging: false,
  draggable: true,
  selectable: true,
  deletable: false,
  zIndex: 0,
  isConnectable: true,
  positionAbsoluteX: 0,
  positionAbsoluteY: 0,
};

const renderNode = () =>
  render(
    <ReactFlowProvider>
      <GatewayNode {...props} />
    </ReactFlowProvider>,
  );

describe("GatewayNode", () => {
  it("names the gateway kind in plain words with the container name under it", () => {
    renderNode();
    expect(screen.getByText("Cloudflare tunnel")).toBeInTheDocument();
    expect(screen.getByText("cloudflared-local")).toBeInTheDocument();
  });

  it("renders one row per route reading service:port with the address when known", () => {
    renderNode();
    expect(screen.getByText("hello-api:3000")).toBeInTheDocument();
    expect(screen.getByText("(172.18.0.4:3000)")).toBeInTheDocument();
    expect(screen.getByText("web:80")).toBeInTheDocument();
    expect(screen.queryByText(/\(:80\)/)).not.toBeInTheDocument();
  });

  it("gives every row its own wire in and wire out", () => {
    const { container } = renderNode();
    expect(container.querySelector('[data-handleid="in-e1"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="e1"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="in-e2"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="e2"]')).toBeInTheDocument();
  });
});
