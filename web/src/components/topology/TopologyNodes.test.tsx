import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type Node, type NodeProps } from "@xyflow/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { ExternalNode } from "@/components/topology/ExternalNode";
import { GatewayNode } from "@/components/topology/GatewayNode";
import { HostnameNode } from "@/components/topology/HostnameNode";
import { NetworkNode } from "@/components/topology/NetworkNode";
import { ServiceNode } from "@/components/topology/ServiceNode";
import {
  ExternalLabel,
  ServiceStatus,
  type ExternalNode as ExternalNodeType,
  type GatewayNode as GatewayNodeType,
  type HostnameNode as HostnameNodeType,
  type NetworkNode as NetworkNodeType,
  type ServiceNode as ServiceNodeType,
} from "@/models/Topology";

const nodeProps = <T extends Node>(type: string, data: T["data"], overrides: Partial<NodeProps<T>> = {}): NodeProps<T> =>
  ({
    id: `${type}-1`,
    data,
    selected: false,
    type,
    dragging: false,
    draggable: true,
    selectable: true,
    deletable: true,
    zIndex: 0,
    isConnectable: true,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    ...overrides,
  }) as NodeProps<T>;

const renderNode = (node: ReactElement) => render(<ReactFlowProvider>{node}</ReactFlowProvider>);

const serviceData: ServiceNodeType["data"] = {
  service_id: "svc-api",
  name: "api-gateway",
  runtime: "go",
  url: "https://api.example.com",
  status: ServiceStatus.Healthy,
  replicas: 3,
};

const externalData: ExternalNodeType["data"] = { name: "Cloudflare", label: ExternalLabel.Tunnel, url: "https://example.com" };

describe("ServiceNode", () => {
  it("renders the name, url, status badge, replicas, volume and the runner and strategy row", () => {
    renderNode(
      <ServiceNode
        {...nodeProps<ServiceNodeType>("service", { ...serviceData, status: ServiceStatus.Failed, volume: "/data/pg", target: "instance", strategy: "run" })}
      />,
    );
    expect(screen.getByText("api-gateway")).toBeInTheDocument();
    expect(screen.getByRole("link")).toHaveTextContent("https://api.example.com");
    expect(screen.getByText(ServiceStatus.Failed)).toBeInTheDocument();
    expect(screen.getByText("3 replicas")).toBeInTheDocument();
    expect(screen.getByText("/data/pg")).toBeInTheDocument();
    expect(screen.getByText("instance · run")).toBeInTheDocument();
  });

  it("omits the url link and the replicas footer when the service has neither", () => {
    renderNode(<ServiceNode {...nodeProps<ServiceNodeType>("service", { ...serviceData, url: undefined, replicas: undefined })} />);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.queryByText(/replicas/)).not.toBeInTheDocument();
  });
});

describe("ExternalNode", () => {
  it("renders the name, label, and url, and omits the link without a url", () => {
    const { unmount } = renderNode(<ExternalNode {...nodeProps<ExternalNodeType>("external", externalData)} />);
    expect(screen.getByText("Cloudflare")).toBeInTheDocument();
    expect(screen.getByText(ExternalLabel.Tunnel)).toBeInTheDocument();
    expect(screen.getByText("https://example.com")).toBeInTheDocument();
    unmount();

    renderNode(<ExternalNode {...nodeProps<ExternalNodeType>("external", { name: "Cloudflare", label: ExternalLabel.Tunnel })} />);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});

describe("NetworkNode", () => {
  it("names the network and marks it as a network", () => {
    renderNode(<NetworkNode {...nodeProps<NetworkNodeType>("network", { name: "app-net" })} />);
    expect(screen.getByText("app-net")).toBeInTheDocument();
    expect(screen.getByText("network")).toBeInTheDocument();
  });
});

describe("HostnameNode", () => {
  it("links out to the hostname", () => {
    renderNode(
      <HostnameNode
        {...nodeProps<HostnameNodeType>("hostname", { hostname: "llmtested.example.com", kind: "tunnel" }, { draggable: false, selectable: false, deletable: false, isConnectable: false })}
      />,
    );
    const link = screen.getByRole("link", { name: "llmtested.example.com" });
    expect(link).toHaveAttribute("href", "https://llmtested.example.com");
    expect(link).toHaveAttribute("target", "_blank");
  });
});

describe("GatewayNode", () => {
  const gateway = () =>
    renderNode(
      <GatewayNode
        {...nodeProps<GatewayNodeType>(
          "gateway",
          {
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
          { deletable: false },
        )}
      />,
    );

  it("names the gateway kind in plain words with the container name under it", () => {
    gateway();
    expect(screen.getByText("Cloudflare tunnel")).toBeInTheDocument();
    expect(screen.getByText("cloudflared-local")).toBeInTheDocument();
  });

  it("renders one row per route reading service:port with the address when known", () => {
    gateway();
    expect(screen.getByText("hello-api:3000")).toBeInTheDocument();
    expect(screen.getByText("(172.18.0.4:3000)")).toBeInTheDocument();
    expect(screen.getByText("web:80")).toBeInTheDocument();
    expect(screen.queryByText(/\(:80\)/)).not.toBeInTheDocument();
  });

  it("gives every row its own wire in and wire out", () => {
    const { container } = gateway();
    expect(container.querySelector('[data-handleid="in-e1"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="e1"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="in-e2"]')).toBeInTheDocument();
    expect(container.querySelector('[data-handleid="e2"]')).toBeInTheDocument();
  });
});

describe("a selected node", () => {
  it.each([
    ["service", <ServiceNode key="s" {...nodeProps<ServiceNodeType>("service", serviceData, { selected: true })} />],
    ["external", <ExternalNode key="e" {...nodeProps<ExternalNodeType>("external", externalData, { selected: true })} />],
    ["network", <NetworkNode key="n" {...nodeProps<NetworkNodeType>("network", { name: "app-net" }, { selected: true })} />],
  ])("carries the selection on a %s node", (_type, node) => {
    const { container } = renderNode(node);
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
