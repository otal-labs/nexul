import type { Exposure, Gateway } from "@/models/DNS";
import type { Container, Stack } from "@/models/Stack";
import {
  NodeType,
  RelationKind,
  type GatewayNode,
  type HostnameNode,
  type NetworkGroup,
  type RelationEdge,
  type ServiceNode,
  type TopologyNode,
} from "@/models/Topology";

export interface Wiring {
  // Stored nodes: service nodes enriched with machine and strategy; a gateway's service node becomes a gateway node.
  nodes: TopologyNode[];
  // Derived hostname pills; they live outside the store, so the canvas keeps their positions and measurements.
  hosts: HostnameNode[];
  edges: RelationEdge[];
  networks: NetworkGroup[];
}

export const hostnameNodeId = (exposureId: string) => `host-${exposureId}`;
export const routeInHandle = (exposureId: string) => `in-${exposureId}`;

// A bare wire: the gateway row it touches already says everything.
const routeEdge = (id: string, source: string, target: string, handles: Partial<Pick<RelationEdge, "sourceHandle" | "targetHandle">> = {}): RelationEdge => ({
  id,
  source,
  target,
  type: "route",
  data: { kind: RelationKind.ConnectsTo },
  selectable: false,
  deletable: false,
  ...handles,
});

// Without a gateway node the port has nowhere else to live, so it labels the direct wire.
const directEdge = (id: string, source: string, target: string, port: number): RelationEdge => ({
  id,
  source,
  target,
  type: "relation",
  data: { kind: RelationKind.ConnectsTo, label: `:${port}` },
  selectable: false,
  deletable: false,
});

const append = (map: Map<string, string[]>, key: string, value: string) =>
  map.set(key, [...(map.get(key) ?? []), value]);

// The canvas stays a map (ADR 0033): nothing here is persisted. Networks, hostname pills, gateway rows, and route wires
// are read off the observed containers, gateways, and exposures every render, so the drawing can never disagree
// with what deploys. Containers replace the old per-project service definitions (spec §2): a service node's
// `service_id` is a container id, and exposures target one directly instead of a service name.
export const deriveWiring = (
  storedNodes: TopologyNode[],
  containers: Container[],
  stacks: Stack[],
  gateways: Gateway[],
  exposures: Exposure[],
): Wiring => {
  const containerById = new Map(containers.map((c) => [c.id, c]));
  const stackById = new Map(stacks.map((s) => [s.id, s]));
  // Canvases and gateways saved before containers existed still name the stack (then the "service") where a
  // container id now goes; a one-container stack resolves to that container so its wires and page keep working.
  const resolveContainerId = (id: string): string => {
    if (containerById.has(id)) return id;
    return containers.find((c) => c.stack_id === id)?.id ?? id;
  };
  const nodeByContainerId = new Map<string, ServiceNode>();
  const stored = storedNodes.map((n) => {
    if (n.type !== NodeType.Service) return n;
    const resolved: ServiceNode = { ...n, data: { ...n.data, service_id: resolveContainerId(n.data.service_id) } };
    nodeByContainerId.set(resolved.data.service_id, resolved);
    return resolved;
  });
  const gatewayByContainerId = new Map(
    gateways.filter((g) => g.service_id).map((g) => [resolveContainerId(g.service_id!), g]),
  );

  const networks = new Map<string, string[]>();
  const nodes: TopologyNode[] = stored.map((n) => {
    if (n.type !== NodeType.Service) return n;
    const container = containerById.get(n.data.service_id);
    const stack = container ? stackById.get(container.stack_id) : undefined;
    const gateway = gatewayByContainerId.get(n.data.service_id);
    const extras = { target: stack?.machine, strategy: stack?.strategy, stackId: container?.stack_id };
    // A gateway joins every network it serves, so boxing it would stack every box on top of it; it stays outside
    // and its card lists the networks instead.
    if (!gateway) {
      for (const net of container?.networks ?? []) append(networks, net.name, n.id);
      return { ...n, data: { ...n.data, ...extras } };
    }
    const joined = [...new Set([...(container?.networks ?? []).map((net) => net.name), ...(gateway.networks ?? [gateway.docker_network])])];
    const routes = exposures
      .filter((e) => e.gateway_id === gateway.id)
      .map((e) => {
        const targetContainer = e.service_id ? containerById.get(e.service_id) : undefined;
        const targetNode = e.service_id ? nodeByContainerId.get(e.service_id) : undefined;
        return {
          id: e.id,
          hostname: e.hostname,
          service: targetContainer?.name ?? e.service,
          port: e.port,
          address: targetNode?.data.address,
        };
      });
    const view: GatewayNode = {
      ...n,
      type: "gateway",
      data: { ...n.data, ...extras, kind: gateway.kind, routes, networks: joined },
    };
    return view;
  });

  const hosts: HostnameNode[] = [];
  const edges: RelationEdge[] = [];
  for (const e of exposures) {
    const gateway = gateways.find((g) => g.id === e.gateway_id);
    const gatewayNode = gateway?.service_id ? nodeByContainerId.get(resolveContainerId(gateway.service_id)) : undefined;
    const target = e.service_id ? nodeByContainerId.get(e.service_id) : undefined;
    const hostId = hostnameNodeId(e.id);
    hosts.push({
      id: hostId,
      type: "hostname",
      position: { x: 0, y: 0 },
      data: { hostname: e.hostname, kind: gateway?.kind },
    });
    if (gatewayNode) {
      edges.push(routeEdge(`route-${e.id}-in`, hostId, gatewayNode.id, { targetHandle: routeInHandle(e.id) }));
      if (target && target.id !== gatewayNode.id) edges.push(routeEdge(`route-${e.id}-out`, gatewayNode.id, target.id, { sourceHandle: e.id }));
      continue;
    }
    if (target) edges.push(directEdge(`route-${e.id}`, hostId, target.id, e.port));
  }

  return {
    nodes,
    hosts,
    edges,
    networks: [...networks].map(([name, memberIds]) => ({ name, memberIds })).sort((a, b) => a.name.localeCompare(b.name)),
  };
};
