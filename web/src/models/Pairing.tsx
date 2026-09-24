import { z } from "zod";

// The bearer token never reaches the browser — only the metadata settings needs.
export interface Computer {
  id: string;
  name: string;
  server_url: string;
  token_expires_at: string;
  kind: string;
  harness_version: string;
  created_at: string;
  updated_at: string;
  // Absent for a computer paired by URL.
  tunnel?: ComputerTunnel;
}

// A computer's own personal access token, "Nexul MCP on <computer>"; never carries the secret.
export interface MCPToken {
  id: string;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string | null;
}

// Only the mint response carries the raw token.
export interface MintedMCPToken extends MCPToken {
  token: string;
}

export interface ComputerTunnel {
  tunnel_id: string;
  hostname: string;
}

// A computer tunnel's two checks; tunnel is Cloudflare's connector status: inactive, healthy, degraded, or down.
export interface TunnelStatus {
  tunnel: string;
  harness_reachable: boolean;
  harness_version?: string;
}

// Go's zero time: a computer tunnel holds no session until T3 Code pairs over its hostname.
export const stillPairing = (computer: Computer) => !(Date.parse(computer.token_expires_at) > 0);

export const tunnelOnline = (status: TunnelStatus) => status.tunnel === "healthy" || status.tunnel === "degraded";
export const tunnelConnected = (status: TunnelStatus) => tunnelOnline(status) && status.harness_reachable;

// The pair-a-computer dialog's steps, in order.
export const PAIRING_STEPS = [
  { value: "connect", label: "Connect" },
  { value: "pair", label: "Pair T3 Code" },
  { value: "setup", label: "Set up" },
] as const;

export type PairingStep = (typeof PAIRING_STEPS)[number]["value"];

// What the instance needs before any computer can be reached through a tunnel; mirrors pairing.PrerequisiteReason.
export type TunnelPrerequisite = "cloudflare_not_connected" | "zero_trust_disabled";

// The Go default in pairing.DefaultT3CodePort; the port T3 Code serves on unless started with another.
export const DEFAULT_T3_CODE_PORT = 3773;

export const CreateComputerTunnelFormSchema = z.object({
  name: z.string().trim().min(1, "Name this computer"),
  port: z.coerce.number<number>().int("Enter a whole port number").min(1, "Enter a port between 1 and 65535").max(65535, "Enter a port between 1 and 65535"),
});

export type CreateComputerTunnelFormData = z.infer<typeof CreateComputerTunnelFormSchema>;

// Labels for the harness kinds a computer can be paired with; keys match the Go harness.Kind values.
export const HARNESS_LABELS: Record<string, string> = { t3code: "T3 Code" };
export const harnessLabel = (kind: string) => HARNESS_LABELS[kind] ?? kind;

// A user's pairing defaults for chat contexts with no linked project.
export interface PairingDefaults {
  default_computer_id?: string;
  fallback_project_id?: string;
  provider?: string;
  model?: string;
}

// An empty computer_id means never linked; chat mentions fall back to the user's own pairing defaults.
export interface ProjectLink {
  project_id?: string;
  computer_id?: string;
  harness_project_id?: string;
  provider?: string;
  model?: string;
}

export interface HarnessProject {
  id: string;
  title: string;
}

export interface HarnessProviderModel {
  slug: string;
  name: string;
  is_default?: boolean;
}

// A usable provider instance; id is the instanceId CreateThread routes on, driver the kind setup is confirmed under.
export interface HarnessProvider {
  id: string;
  driver: string;
  name: string;
  models: HarnessProviderModel[];
  needs_setup: boolean;
}

export const PairComputerFormSchema = z.object({
  name: z.string().trim().min(1, "Computer name is required"),
  server_url: z.string().trim().url("Enter a valid URL, e.g. https://your-t3-host:port"),
  token: z.string().trim().min(1, "Paste the one-time pairing token"),
});

export type PairComputerFormData = z.infer<typeof PairComputerFormSchema>;

// The inputs the pairing routes key a failure under, so it shows on the field that caused it.
export const PAIR_FIELDS = ["name", "server_url", "token"] as const satisfies readonly (keyof PairComputerFormData)[];
export type PairField = (typeof PAIR_FIELDS)[number];

export const PairingDefaultsFormSchema = z.object({
  default_computer_id: z.string(),
  fallback_project_id: z.string(),
  provider: z.string(),
  model: z.string(),
});

export type PairingDefaultsFormData = z.infer<typeof PairingDefaultsFormSchema>;

export const ProjectLinkFormSchema = z.object({
  computer_id: z.string().trim().min(1, "Pick a computer"),
  harness_project_id: z.string().trim().min(1, "T3 project id is required"),
  provider: z.string().trim(),
  model: z.string().trim(),
});

export type ProjectLinkFormData = z.infer<typeof ProjectLinkFormSchema>;

// The settings UI's threshold for flagging a computer as about to act like unpaired.
export const EXPIRY_WARNING_DAYS = 5;

// Whether the caller can run a play right now: the resolved computer, provider, and model, or a reason it
// can't, joined against presence. provider/model travel here so the run dialog's pill can preselect them
// without a second call to the same resolve endpoint.
export type HarnessReadiness =
  | { state: "ready"; computerId: string; provider: string; model: string }
  | { state: "unpaired" | "expired" | "no_harness_project" | "no_default_computer" | "offline"; message: string };

// Copy for every non-ready state; shared by the hook's join and the settings readiness line. Says harness, never computer.
export const HARNESS_READINESS_COPY: Record<Exclude<HarnessReadiness["state"], "ready">, string> = {
  unpaired: "Pair a harness in Settings to run plays",
  expired: "Your harness pairing has expired, re-pair it in Settings",
  no_harness_project: "Pick a harness project for this project, or set a fallback in Settings",
  no_default_computer: "Several harnesses are paired, pick a default in Settings",
  offline: "Your harness is offline",
};
