import { z } from "zod";

export const RecordTypes = ["A", "AAAA", "CNAME", "TXT"] as const;
export type RecordType = (typeof RecordTypes)[number];

export interface Zone {
  id: string;
  name: string;
  status?: string;
}

export interface DnsRecord {
  id: string;
  zone_id: string;
  type: RecordType;
  name: string;
  content: string;
  ttl: number;
}

export const InstanceRecordFormSchema = z.object({
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
  type: z.enum(RecordTypes),
  target: z.string().trim().min(1, "Server address or tunnel hostname is required"),
});

export type InstanceRecordFormData = z.infer<typeof InstanceRecordFormSchema>;

// The three ways traffic can reach the instance; the DNS setup stepper's first choice.
export const EntryPaths = {
  Bare: "bare",
  Tunnel: "tunnel",
  Proxy: "proxy",
} as const;

export type EntryPath = (typeof EntryPaths)[keyof typeof EntryPaths];

export interface EntryPathOption {
  value: EntryPath;
  label: string;
  description: string;
  // What the hostname step explains once this path is chosen.
  stepDescription: string;
}

export const entryPathOptions: EntryPathOption[] = [
  {
    value: EntryPaths.Bare,
    label: "Bare public address",
    description: "Point the instance hostname at the server directly with an A or AAAA record.",
    stepDescription: "A record under your zone points the instance hostname at the server's address.",
  },
  {
    value: EntryPaths.Tunnel,
    label: "Cloudflare tunnel",
    description: "No open ports. cloudflared runs as a service and routes the hostname through it.",
    stepDescription: "Routes the hostname into the connected tunnel and creates its DNS record.",
  },
  {
    value: EntryPaths.Proxy,
    label: "Reverse proxy",
    description: "Traefik runs as a service on the server and routes hostnames to containers.",
    stepDescription: "Nexul deploys Traefik on your runner and points the instance record at the server.",
  },
];

// The tunnel rung's output: what the hostname rung routes into and where cloudflared runs.
export interface TunnelDeployment {
  tunnelId: string;
  tunnelName: string;
  serviceId: string;
  target: string;
}

// What the "Go live" step shows once the hostname step succeeded.
export interface DnsSetupResult {
  headline: string;
  detail: string;
}

// Mirrors the server's recordNameFor: "@" at the apex, the label under the zone, or null when the host isn't in it.
export const instanceRecordName = (instanceUrl: string, zone: string): string | null => {
  let host = "";
  try {
    host = new URL(instanceUrl).hostname.toLowerCase();
  } catch {
    return null;
  }
  const z = zone.toLowerCase();
  if (!host || !z) return null;
  if (host === z) return "@";
  if (host.endsWith(`.${z}`)) return host.slice(0, -(z.length + 1));
  return null;
};

// Cloudflare's subdomain + zone split: an empty subdomain means the zone apex.
export const fullHostname = (subdomain: string, zone: string): string =>
  subdomain.trim() ? `${subdomain.trim()}.${zone}` : zone;

// Preselects the obvious choice on a fresh install (one zone, one project, one runner) without an effect.
export const soleItem = <T,>(items: T[]): T | undefined => (items.length === 1 ? items[0] : undefined);

export interface Tunnel {
  id: string;
  name: string;
  account_id: string;
  status: string;
  hostname?: string;
  zone_id?: string;
  zone?: string;
  record_id?: string;
  service?: string;
  agent_service_id?: string;
  created_at: string;
  updated_at: string;
}

// A Nexul-deployed "exit node" giving one docker network internet reachability; one per network.
export const GatewayKinds = ["tunnel", "proxy"] as const;
export type GatewayKind = (typeof GatewayKinds)[number];

export interface Gateway {
  id: string;
  kind: GatewayKind;
  docker_network: string;
  // Every docker network this gateway's container has joined (home network plus anything an exposure added).
  networks?: string[];
  // The machine the gateway's backing container deploys to; an exposure's target must share it (spec §7).
  machine?: string;
  service_id?: string;
  service_name?: string;
  tunnel_id?: string;
  zone_id: string;
  zone: string;
  server_address?: string;
  created_at: string;
  updated_at: string;
}

export const CreateGatewayFormSchema = z
  .object({
    kind: z.enum(GatewayKinds),
    docker_network: z.string().trim().min(1, "Docker network is required"),
    zone_id: z.string().min(1, "Choose a zone"),
    zone: z.string().min(1, "Choose a zone"),
    tunnel_id: z.string().trim(),
    server_address: z.string().trim(),
    project_id: z.string().min(1, "Choose a project"),
    target: z.string().trim().min(1, "Runner is required"),
  })
  .superRefine((val, ctx) => {
    if (val.kind === "tunnel" && !val.tunnel_id) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["tunnel_id"], message: "Choose a tunnel" });
    }
    if (val.kind === "proxy" && !val.server_address) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["server_address"],
        message: "Server address is required for a proxy gateway",
      });
    }
  });

export type CreateGatewayFormData = z.infer<typeof CreateGatewayFormSchema>;

// A hostname routed through a gateway to a container.
export interface Exposure {
  id: string;
  gateway_id: string;
  hostname: string;
  // The container this exposure routes to (services.id); the primary target going forward.
  service_id?: string;
  // Legacy name column: kept as a display fallback for exposures created before the container rename.
  service: string;
  port: number;
  zone_id: string;
  zone: string;
  record_id?: string;
  created_at: string;
  updated_at: string;
}

// The gateway is resolved (reused or provisioned) by the backend now — this only picks the container, port,
// hostname, and zone (spec §7).
export const ExposeServiceFormSchema = z.object({
  hostname: z.string().trim().min(1, "Hostname is required"),
  service_id: z.string().min(1, "Choose a container"),
  port: z.coerce.number<number>().int().positive("Container port must be positive"),
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
});

export type ExposeServiceFormData = z.infer<typeof ExposeServiceFormSchema>;

