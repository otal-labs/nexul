import { render, screen } from "@testing-library/react";
import { ReactFlowProvider, type NodeProps } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { ServiceNode } from "@/components/topology/ServiceNode";
import { ServiceStatus, type ServiceNode as ServiceNodeType } from "@/models/Topology";

type Props = NodeProps<ServiceNodeType>;

const baseData = {
  service_id: "svc-api",
  name: "api-gateway",
  runtime: "go",
  url: "https://api.example.com",
  status: ServiceStatus.Healthy,
  replicas: 3,
};

const makeProps = (overrides: Partial<Props> = {}): Props => ({
  id: "svc-api",
  data: baseData,
  selected: false,
  type: "service",
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
      <ServiceNode {...makeProps(overrides)} selected={selected} />
    </ReactFlowProvider>,
  );

describe("ServiceNode", () => {
  it("renders the service name and url", () => {
    renderNode();
    expect(screen.getByText("api-gateway")).toBeInTheDocument();
    expect(screen.getByText("https://api.example.com")).toBeInTheDocument();
  });

  it("renders the status badge", () => {
    renderNode({ data: { ...baseData, status: ServiceStatus.Failed } });
    expect(screen.getByText(ServiceStatus.Failed)).toBeInTheDocument();
  });

  it("renders the replicas footer when present", () => {
    renderNode();
    expect(screen.getByText("3 replicas")).toBeInTheDocument();
  });

  it("omits the replicas footer when absent", () => {
    renderNode({ data: { ...baseData, replicas: undefined } });
    expect(screen.queryByText(/replicas/)).not.toBeInTheDocument();
  });

  it("renders the volume footer when present", () => {
    renderNode({ data: { ...baseData, volume: "/data/pg" } });
    expect(screen.getByText("/data/pg")).toBeInTheDocument();
  });

  it("shows the runner and strategy row when wired in", () => {
    renderNode({ data: { ...baseData, target: "instance", strategy: "run" } });
    expect(screen.getByText("instance · run")).toBeInTheDocument();
  });

  it("does not render a url link when absent", () => {
    renderNode({ data: { ...baseData, url: undefined } });
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("applies the selection ring when selected", () => {
    const { container } = renderNode({}, true);
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
