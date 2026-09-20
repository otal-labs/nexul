import type { Edge, Node } from "@xyflow/react";
import { z } from "zod";

import type { GatewayKind } from "@/models/DNS";

export const RelationKind = {
  DependsOn: "depends_on",
  ConnectsTo: "connects_to",
  Mounts: "mounts",
} as const;

export type RelationKind = (typeof RelationKind)[keyof typeof RelationKind];

export const ServiceStatus = {
  Healthy: "healthy",
  Running: "running",
  Stopped: "stopped",
  Failed: "failed",
} as const;

export type ServiceStatus = (typeof ServiceStatus)[keyof typeof ServiceStatus];

export const NodeType = {
  Service: "service",
  Network: "network",
  External: "external",
} as const;

export type NodeType = (typeof NodeType)[keyof typeof NodeType];

export const ExternalLabel = {
  Domain: "domain",
  Tunnel: "tunnel",
  Proxy: "proxy",
  Database: "database",
  Api: "api",
  Other: "other",
} as const;

export type ExternalLabel = (typeof ExternalLabel)[keyof typeof ExternalLabel];

export const PositionSchema = z.object({ x: z.number().finite(), y: z.number().finite() });

export const ServiceNodeDataSchema = z.object({
  service_id: z.string().min(1),
  name: z.string().min(1),
  runtime: z.string().optional(),
  url: z.string().optional(),
  status: z.enum([
    ServiceStatus.Healthy,
    ServiceStatus.Running,
    ServiceStatus.Stopped,
    ServiceStatus.Failed,
  ]),
  replicas: z.number().int().nonnegative().optional(),
  volume: z.string().optional(),
  // The container's address on its docker network, reported by the runner when the service went healthy.
  address: z.string().optional(),
});

export const NetworkNodeDataSchema = z.object({
  name: z.string().min(1),
});

export const ExternalNodeDataSchema = z.object({
  name: z.string().min(1),
  label: z.enum([
    ExternalLabel.Domain,
    ExternalLabel.Tunnel,
    ExternalLabel.Proxy,
    ExternalLabel.Database,
    ExternalLabel.Api,
    ExternalLabel.Other,
  ]),
  url: z.string().optional(),
});

export const ServiceNodeSchema = z.object({
  id: z.string().min(1),
  type: z.literal(NodeType.Service),
  position: PositionSchema,
  data: ServiceNodeDataSchema,
});

export const NetworkNodeSchema = z.object({
  id: z.string().min(1),
  type: z.literal(NodeType.Network),
  position: PositionSchema,
  data: NetworkNodeDataSchema,
});

export const ExternalNodeSchema = z.object({
  id: z.string().min(1),
  type: z.literal(NodeType.External),
  position: PositionSchema,
  data: ExternalNodeDataSchema,
});

export const TopologyNodeSchema = z.discriminatedUnion("type", [
  ServiceNodeSchema,
  NetworkNodeSchema,
  ExternalNodeSchema,
]);

export const RelationEdgeDataSchema = z.object({
  kind: z.enum([RelationKind.DependsOn, RelationKind.ConnectsTo, RelationKind.Mounts]),
  // Overrides the kind's label on derived route edges (":3000"); never stored.
  label: z.string().optional(),
});

export const RelationEdgeSchema = z.object({
  id: z.string().min(1),
  source: z.string().min(1),
  target: z.string().min(1),
  type: z.literal("relation"),
  data: RelationEdgeDataSchema,
});

// Where the owner last left the camera; saved with the map so a reload opens the canvas where it was.
export const ViewportSchema = z.object({ x: z.number(), y: z.number(), zoom: z.number() });

export const CanvasSchema = z.object({
  schema_version: z.literal(2),
  nodes: z.array(TopologyNodeSchema),
  edges: z.array(RelationEdgeSchema),
  viewport: ViewportSchema.optional(),
});

export type Canvas = z.infer<typeof CanvasSchema>;
export type Viewport = z.infer<typeof ViewportSchema>;
export type ServiceNodeData = z.infer<typeof ServiceNodeDataSchema>;
export type NetworkNodeData = z.infer<typeof NetworkNodeDataSchema>;
export type ExternalNodeData = z.infer<typeof ExternalNodeDataSchema>;
export type TopologyNodeData = ServiceNodeData | NetworkNodeData | ExternalNodeData;
export type RelationEdgeData = z.infer<typeof RelationEdgeDataSchema>;

// View-only fields wired onto a service node at render time from the stacks/containers list; never persisted.
// Type aliases, not interfaces: React Flow node data must be assignable to Record<string, unknown>.
export type ServiceNodeExtras = {
  target?: string | undefined;
  strategy?: string | undefined;
  // The owning stack's id (service_id is now a container id, spec §2) — lets a node click open its stack page.
  stackId?: string | undefined;
};

// A hostname routed to a service through a gateway; derived from exposures at render time, never persisted.
export type HostnameNodeData = {
  hostname: string;
  kind?: GatewayKind | undefined;
};

// One route a gateway forwards: the row reads "→ service:port (address:port)" and owns a wire in and a wire out.
export type RouteRow = {
  id: string;
  hostname: string;
  service: string;
  port: number;
  address?: string | undefined;
};

// A gateway's service node, rendered as the hub its routes pass through; same id as the stored service node.
export type GatewayNodeData = ServiceNodeData & {
  target?: string | undefined;
  stackId?: string | undefined;
  kind: GatewayKind;
  routes: RouteRow[];
  // Every docker network the gateway is on; it sits outside the network boxes, so the card names them instead.
  networks: string[];
};

export type ServiceNode = Node<ServiceNodeData & ServiceNodeExtras, "service">;
export type NetworkNode = Node<NetworkNodeData, "network">;
export type ExternalNode = Node<ExternalNodeData, "external">;
export type HostnameNode = Node<HostnameNodeData, "hostname">;
export type GatewayNode = Node<GatewayNodeData, "gateway">;
export type TopologyNode = ServiceNode | NetworkNode | ExternalNode | HostnameNode | GatewayNode;
// "relation" edges are stored and hand-drawn; "route" edges are derived wires and never persisted.
export type RelationEdge = Edge<RelationEdgeData, "relation" | "route">;

// A docker network and the service nodes on it; drawn as a dashed box around its members.
export interface NetworkGroup {
  name: string;
  memberIds: string[];
}
