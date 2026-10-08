import { z } from "zod";
import type { DnsStepState } from "@/components/dns/DnsStep";
import type { CredentialCheck } from "@/models/Connectors";

export const RecordTypes = ["A", "AAAA", "CNAME", "TXT"] as const;
export type RecordType = (typeof RecordTypes)[number];

export interface Zone {
  id: string;
  name: string;
  status?: string;
  account_id?: string;
  account_name?: string;
}

export interface CloudflareAccount {
  id: string;
  name: string;
}

// The accounts owning the zones, once each; a tunnel only serves hostnames in its own account's zones.
export const zoneAccounts = (zones: Zone[]): CloudflareAccount[] => {
  const byId = new Map<string, CloudflareAccount>();
  for (const z of zones) {
    if (!z.account_id || byId.has(z.account_id)) continue;
    byId.set(z.account_id, { id: z.account_id, name: z.account_name ?? z.account_id });
  }
  return [...byId.values()];
};

// The zones a tunnel in accountId can serve; an unknown account (an older tunnel) keeps every zone.
export const zonesInAccount = (zones: Zone[], accountId: string): Zone[] =>
  zones.filter((z) => !accountId || !z.account_id || z.account_id === accountId);

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

// The ways traffic can reach the instance; the DNS setup stepper's and the first-run domain step's first choice.
export const EntryPaths = {
  Bare: "bare",
  Tunnel: "tunnel",
  Proxy: "proxy",
  OwnHttps: "https",
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
    stepDescription: "Nexul deploys Traefik on a machine and points the instance record at the server.",
  },
];

// First run offers only paths that end in HTTPS, which the GitHub callback needs; a bare record stays on the DNS page.
export const setupEntryPathOptions: EntryPathOption[] = [
  {
    value: EntryPaths.Tunnel,
    label: "Cloudflare tunnel",
    description: "No open ports. Needs a Cloudflare API token; Cloudflare issues the certificate.",
    stepDescription: "Connect Cloudflare, run cloudflared on this server, then route a hostname into it.",
  },
  {
    value: EntryPaths.Proxy,
    label: "Reverse proxy",
    description: "Nexul runs Traefik on ports 80 and 443 and gets a Let's Encrypt certificate.",
    stepDescription: "Point the domain at this server, then Nexul deploys Traefik and waits for the certificate.",
  },
  {
    value: EntryPaths.OwnHttps,
    label: "I already have HTTPS",
    description: "Your own proxy already serves an https address that forwards to this server.",
    stepDescription: "Nexul checks the address answers as this instance over HTTPS.",
  },
];

export const stepState = (unlocked: boolean, done: boolean): DnsStepState => {
  if (!unlocked) return "upcoming";
  if (done) return "done";
  return "active";
};

// The tunnel rung's output: what the hostname rung routes into and where cloudflared runs.
export interface TunnelDeployment {
  tunnelId: string;
  tunnelName: string;
  // The Cloudflare account the tunnel lives in; its hostname must be in one of that account's zones.
  accountId: string;
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
  let host: string;
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
    target: z.string().trim().min(1, "Machine is required"),
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

// The rows of the ticker that runs once a hostname is routed into a tunnel, one verify request each.
export const TUNNEL_ROUTE_CHECKS: CredentialCheck[] = [
  { key: "ingress", label: "Route into the tunnel", why: "cloudflared forwards the hostname to its local service." },
  { key: "record", label: "DNS record", why: "A proxied CNAME points the hostname at the tunnel." },
  {
    key: "reachable",
    label: "Answers over HTTPS",
    why: "Asked from this server through Cloudflare; a new record can take a minute.",
  },
];
