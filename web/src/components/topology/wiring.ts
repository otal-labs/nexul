import { ROUTE_IN } from "@/components/topology/canvasStyles";
import { routeRowHeights } from "@/components/topology/geometry";
import type { Exposure, Gateway } from "@/models/DNS";
import type { Container, Stack } from "@/models/Stack";
import {
  NodeType,
  RelationKind,
  ServiceStatus,
  type AttentionStatus,
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

// A service that needs attention colours the whole path that reaches it; a healthy one leaves it neutral.
const toneOf = (status: string | undefined): AttentionStatus | undefined => {
  if (status === ServiceStatus.Failed || status === ServiceStatus.Stopped) return status;
  return undefined;
};

const withTone = (tone: AttentionStatus | undefined) => (tone ? { tone } : {});

// A bare wire: the gateway row it touches already says everything.
const routeEdge = (
  id: string,
  source: string,
  target: string,
  exposure: string,
  tone: AttentionStatus | undefined,
  handles: Partial<Pick<RelationEdge, "sourceHandle" | "targetHandle">> = {},
): RelationEdge => ({
  id,
  source,
  target,
  type: "route",
  data: { kind: RelationKind.ConnectsTo, exposure, ...withTone(tone) },
  selectable: false,
  deletable: false,
  ...handles,
});

// Without a gateway node the port has nowhere else to live, so it labels the direct wire.
const directEdge = (id: string, source: string, target: string, port: number, exposure: string, tone: AttentionStatus | undefined): RelationEdge => ({
  id,
  source,
  target,
  targetHandle: ROUTE_IN,
  type: "relation",
  data: { kind: RelationKind.ConnectsTo, label: `:${port}`, exposure, ...withTone(tone) },
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

  // A gateway shared between workspaces routes hostnames into each; only those reaching a service on this canvas show.
  const onCanvas = exposures.filter((e) => e.service_id && nodeByContainerId.has(e.service_id));
  const networks = new Map<string, string[]>();
  const nodes: TopologyNode[] = stored.map((n) => {
    if (n.type !== NodeType.Service) return n;
    const container = containerById.get(n.data.service_id);
    const stack = container ? stackById.get(container.stack_id) : undefined;
    const gateway = gatewayByContainerId.get(n.data.service_id);
    const extras = { target: stack?.machine, strategy: stack?.strategy, stackId: container?.stack_id, serviceName: container?.name };
    // A gateway joins every network it serves, so boxing it would stack every box on top of it; it stays outside
    // and its card lists the networks instead.
    if (!gateway) {
      for (const net of container?.networks ?? []) append(networks, net.name, n.id);
      return { ...n, data: { ...n.data, ...extras } };
    }
    const joined = [...new Set([...(container?.networks ?? []).map((net) => net.name), ...(gateway.networks ?? [gateway.docker_network])])];
    const routes = onCanvas
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
          ...withTone(toneOf(targetNode?.data.status)),
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
  for (const e of onCanvas) {
    const gateway = gateways.find((g) => g.id === e.gateway_id);
    const gatewayNode = gateway?.service_id ? nodeByContainerId.get(resolveContainerId(gateway.service_id)) : undefined;
    const target = e.service_id ? nodeByContainerId.get(e.service_id) : undefined;
    if (!target) continue;
    const hostId = hostnameNodeId(e.id);
    const tone = toneOf(target.data.status);
    hosts.push({
      id: hostId,
      type: "hostname",
      position: { x: 0, y: 0 },
      data: { hostname: e.hostname, kind: gateway?.kind, ...withTone(tone) },
    });
    if (gatewayNode) {
      edges.push(routeEdge(`route-${e.id}-in`, hostId, gatewayNode.id, e.id, tone, { targetHandle: routeInHandle(e.id) }));
      if (target.id !== gatewayNode.id) edges.push(routeEdge(`route-${e.id}-out`, gatewayNode.id, target.id, e.id, tone, { sourceHandle: e.id, targetHandle: ROUTE_IN }));
      continue;
    }
    edges.push(directEdge(`route-${e.id}`, hostId, target.id, e.port, e.id, tone));
  }

  const pillByRoute = new Map(hosts.map((h) => [h.id.slice("host-".length), h.data]));
  for (const n of nodes) {
    if (n.type === "gateway") n.data.rowHeights = routeRowHeights(n.data, pillByRoute);
  }

  // A machine every member of a box runs on is named once in the box header, not on each card.
  const machineOf = new Map(nodes.map((n) => [n.id, n.type === NodeType.Service ? n.data.target : undefined]));
  const groups = [...networks].map(([name, memberIds]): NetworkGroup => {
    const machine = machineOf.get(memberIds[0]!);
    return { name, memberIds, ...(machine && memberIds.every((id) => machineOf.get(id) === machine) ? { machine } : {}) };
  });
  const boxedOn = new Set(groups.flatMap((g) => (g.machine ? g.memberIds : [])));
  return {
    nodes: nodes.map((n) => (n.type === NodeType.Service && boxedOn.has(n.id) ? { ...n, data: { ...n.data, machineInBox: true } } : n)),
    hosts,
    edges,
    networks: groups.sort((a, b) => a.name.localeCompare(b.name)),
  };
};

export interface TopologySummary {
  services: number;
  failed: number;
  stopped: number;
  hostnames: number;
}

// The toolbar's counts: every deployed container (a gateway is one), the ones that need attention, and the public names.
export const summarize = (w: Wiring): TopologySummary => {
  const deployed = w.nodes.filter((n) => n.type === NodeType.Service || n.type === "gateway");
  const count = (status: string) => deployed.filter((n) => n.data.status === status).length;
  return { services: deployed.length, failed: count("failed"), stopped: count("stopped"), hostnames: w.hosts.length };
};

export interface Trace {
  nodes: Set<string>;
  edges: Set<string>;
}

// What a hover lights: every route the node is on, end to end, and the relations that touch it.
export const tracePath = (id: string, edges: RelationEdge[]): Trace => {
  const exposures = new Set(edges.filter((e) => e.data?.exposure && (e.source === id || e.target === id)).map((e) => e.data!.exposure!));
  const lit = edges.filter((e) => (e.data?.exposure && exposures.has(e.data.exposure)) || (!e.data?.exposure && (e.source === id || e.target === id)));
  return {
    nodes: new Set([id, ...lit.flatMap((e) => [e.source, e.target])]),
    edges: new Set(lit.map((e) => e.id)),
  };
};
