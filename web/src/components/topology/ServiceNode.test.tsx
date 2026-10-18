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
  it("renders the name, url, status badge, replicas, volume and the runner and strategy row", () => {
    renderNode({ data: { ...baseData, status: ServiceStatus.Failed, volume: "/data/pg", target: "instance", strategy: "run" } });
    expect(screen.getByText("api-gateway")).toBeInTheDocument();
    expect(screen.getByRole("link")).toHaveTextContent("https://api.example.com");
    expect(screen.getByText(ServiceStatus.Failed)).toBeInTheDocument();
    expect(screen.getByText("3 replicas")).toBeInTheDocument();
    expect(screen.getByText("/data/pg")).toBeInTheDocument();
    expect(screen.getByText("instance · run")).toBeInTheDocument();
  });

  it("omits the url link and the replicas footer when the service has neither", () => {
    renderNode({ data: { ...baseData, url: undefined, replicas: undefined } });
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.queryByText(/replicas/)).not.toBeInTheDocument();
  });

  it("applies the selection ring when selected", () => {
    const { container } = renderNode({}, true);
    expect(container.firstElementChild).toHaveAttribute("data-selected", "true");
  });
});
