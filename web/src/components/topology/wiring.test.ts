import { describe, expect, it } from "vitest";

import { deriveWiring, hostnameNodeId, routeInHandle } from "@/components/topology/wiring";
import type { Exposure, Gateway } from "@/models/DNS";
import { ContainerStatus, DeployStrategy, type Container, type Stack } from "@/models/Stack";
import { ServiceStatus, type TopologyNode } from "@/models/Topology";

const serviceNode = (id: string, name: string, address?: string): TopologyNode => ({
  id,
  type: "service",
  position: { x: 0, y: 0 },
  data: { service_id: id, name, status: ServiceStatus.Healthy, ...(address ? { address } : {}) },
});

const container = (id: string, name: string, network?: string): Container => ({
  id,
  stack_id: "stack-1",
  name,
  declared: {},
  status: ContainerStatus.Healthy,
  ...(network ? { networks: [{ name: network }] } : {}),
});

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "hello",
  slug: "hello",
  machine: "instance",
  strategy: DeployStrategy.Run,
  managed: true,
  created_at: "2026-09-07T00:00:00Z",
  updated_at: "2026-09-07T00:00:00Z",
};

const gateway: Gateway = {
  id: "gw-1",
  kind: "tunnel",
  docker_network: "nexul_default",
  service_id: "svc-cf",
  service_name: "cloudflared-local",
  tunnel_id: "t-1",
  zone_id: "z1",
  zone: "example.com",
  created_at: "2026-09-07T00:00:00Z",
  updated_at: "2026-09-07T00:00:00Z",
};

const exposure: Exposure = {
  id: "exp-1",
  gateway_id: "gw-1",
  hostname: "llmtested.example.com",
  service_id: "svc-api",
  service: "hello-api",
  port: 3000,
  zone_id: "z1",
  zone: "example.com",
  created_at: "2026-09-07T00:00:00Z",
  updated_at: "2026-09-07T00:00:00Z",
};

const stored = [serviceNode("svc-cf", "cloudflared-local"), serviceNode("svc-api", "hello-api", "172.18.0.4")];
const containers = [
  container("svc-cf", "cloudflared-local", "nexul_default"),
  container("svc-api", "hello-api", "nexul_default"),
];
const stacks = [stack];

describe("deriveWiring", () => {
  it("turns the gateway's service node into a gateway node with one route row per exposure", () => {
    const { nodes } = deriveWiring(stored, containers, stacks, [gateway], [exposure]);
    const gw = nodes.find((n) => n.id === "svc-cf");
    expect(gw?.type).toBe("gateway");
    expect(gw?.data).toMatchObject({
      name: "cloudflared-local",
      kind: "tunnel",
      target: "instance",
      stackId: "stack-1",
      routes: [{ id: "exp-1", hostname: "llmtested.example.com", service: "hello-api", port: 3000, address: "172.18.0.4" }],
      networks: ["nexul_default"],
    });
  });

  it("keeps the gateway out of the network boxes and lists every network it joins on its card", () => {
    const spanning = { ...gateway, networks: ["nexul_default", "app_default"] };
    const { nodes, networks } = deriveWiring(stored, containers, stacks, [spanning], [exposure]);
    expect(networks).toEqual([{ name: "nexul_default", memberIds: ["svc-api"] }]);
    const gw = nodes.find((n) => n.id === "svc-cf");
    expect(gw?.type === "gateway" && gw.data.networks).toEqual(["nexul_default", "app_default"]);
  });

  it("wires pill → gateway row → service through the row's own handles, with no labels", () => {
    const { hosts, edges } = deriveWiring(stored, containers, stacks, [gateway], [exposure]);
    expect(hosts).toEqual([expect.objectContaining({ id: hostnameNodeId("exp-1"), data: { hostname: "llmtested.example.com", kind: "tunnel" } })]);
    expect(edges).toEqual([
      expect.objectContaining({ type: "route", source: hostnameNodeId("exp-1"), target: "svc-cf", targetHandle: routeInHandle("exp-1") }),
      expect.objectContaining({ type: "route", source: "svc-cf", sourceHandle: "exp-1", target: "svc-api" }),
    ]);
    expect(edges.every((e) => e.data?.label === undefined)).toBe(true);
  });

  it("groups service nodes by docker network and annotates them with machine, strategy, and stack id", () => {
    const { nodes, networks } = deriveWiring(stored, containers, stacks, [gateway], [exposure]);
    expect(networks).toEqual([{ name: "nexul_default", memberIds: ["svc-api"] }]);
    expect(nodes.find((n) => n.id === "svc-api")?.data).toMatchObject({ target: "instance", strategy: "run", stackId: "stack-1" });
  });

  it("routes straight to the service, port on the wire, when the gateway has no node", () => {
    const { edges } = deriveWiring([serviceNode("svc-api", "hello-api")], containers, stacks, [{ ...gateway, service_id: "" }], [exposure]);
    expect(edges).toEqual([
      expect.objectContaining({ type: "relation", source: hostnameNodeId("exp-1"), target: "svc-api", data: { kind: "connects_to", label: ":3000" } }),
    ]);
  });

  it("keeps one row and one wire per exposure even when several share a target", () => {
    const second = { ...exposure, id: "exp-2", hostname: "api.example.com", port: 3001 };
    const { nodes, hosts, edges } = deriveWiring(stored, containers, stacks, [gateway], [exposure, second]);
    const gw = nodes.find((n) => n.id === "svc-cf");
    expect(gw?.type === "gateway" && gw.data.routes.map((r) => r.port)).toEqual([3000, 3001]);
    expect(hosts).toHaveLength(2);
    expect(edges.filter((e) => e.source === "svc-cf")).toHaveLength(2);
  });

  it("resolves a canvas and gateway saved with stack ids to the stack's container", () => {
    // Before containers existed a node's service_id was the stack id; the container that replaced it has its own.
    const oldNodes = [serviceNode("stack-cf", "cloudflared-local"), serviceNode("stack-1", "hello-api")];
    const migrated = [
      { ...container("c-cf", "cloudflared-local", "nexul_default"), stack_id: "stack-cf" },
      container("c-api", "hello-api", "nexul_default"),
    ];
    const oldGateway = { ...gateway, service_id: "stack-cf" };
    const newExposure = { ...exposure, service_id: "c-api" };
    const { nodes, edges, networks } = deriveWiring(oldNodes, migrated, stacks, [oldGateway], [newExposure]);
    expect(nodes.find((n) => n.id === "stack-1")?.data).toMatchObject({ service_id: "c-api", stackId: "stack-1" });
    expect(nodes.find((n) => n.id === "stack-cf")?.type).toBe("gateway");
    expect(edges).toContainEqual(expect.objectContaining({ source: "stack-cf", sourceHandle: "exp-1", target: "stack-1" }));
    expect(networks).toEqual([{ name: "nexul_default", memberIds: ["stack-1"] }]);
  });

  it("leaves compose services and hand-drawn nodes ungrouped", () => {
    const external: TopologyNode = { id: "ext", type: "external", position: { x: 0, y: 0 }, data: { name: "Stripe", label: "api" } };
    const { networks, nodes } = deriveWiring([serviceNode("svc-api", "hello-api"), external], [container("svc-api", "hello-api")], stacks, [], []);
    expect(networks).toEqual([]);
    expect(nodes).toHaveLength(2);
  });
});
