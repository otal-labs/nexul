// Matches internal/runner's discovery shapes and the snake_case Machine shape of web/src/models/Runner.tsx's
// MachineSchema; kept separate because Runner.tsx and RunnerHooks.tsx serve the Runners page and this only reads
// the same wire shape.

export interface Machine {
  id: string;
  name: string;
  stack_root: string;
  reported_hostname?: string;
  first_seen: string;
  last_seen: string;
}

export interface ContainerNetworkInfo {
  name: string;
  address?: string;
}

export interface DiscoveredContainer {
  name: string;
  image: string;
  status: string;
  labels?: Record<string, string>;
  networks?: ContainerNetworkInfo[];
  ports?: string[];
  // Set on a cloudflared gateway: the tunnel its token connects, and what the Cloudflare connector says about it.
  tunnel_id?: string;
  tunnel?: TunnelInfo;
  // The connector's verdict when the tunnel could not be described (no Cloudflare connector, unknown tunnel).
  tunnel_error?: string;
}

export interface TunnelRoute {
  hostname: string;
  service: string;
}

export interface TunnelRecord {
  name: string;
  content: string;
}

// What the provider knows about a discovered tunnel; tracked means this instance already owns it.
export interface TunnelInfo {
  id: string;
  name: string;
  status: string;
  tracked: boolean;
  routes: TunnelRoute[];
  records: TunnelRecord[];
}

export interface NetworkInfo {
  name: string;
}

export interface StackGroup {
  project: string;
  containers: DiscoveredContainer[];
}

// GroupedDiscovery is POST /api/machines/{id}/discover's response (spec §8).
export interface GroupedDiscovery {
  stacks: StackGroup[];
  standalone: DiscoveredContainer[];
  gateways: DiscoveredContainer[];
  networks: NetworkInfo[];
}

export interface ImportStackGroup {
  project: string;
  containers: string[];
}

export interface ImportRequest {
  project_id: string;
  stacks: ImportStackGroup[];
  standalone: string[];
  gateways: string[];
}

// One ticked gateway's outcome: adopted with the hostnames now on the canvas, or why it could not be.
export interface GatewayAdoption {
  name: string;
  gateway_id?: string;
  exposed: string[];
  // Routed hostnames whose target is not a container Nexul tracks on that machine.
  unmatched: string[];
  error?: string;
}

export interface ImportResult {
  stacks: Array<{ id: string; name: string }>;
  gateways: GatewayAdoption[];
}
