import type { AxiosError } from "axios";
import { z } from "zod";

import type { ApiErrorBody } from "@/api/client";

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

// T3 Code lets no client end its own session, so a removed or replaced pairing's session lives on until it expires.
export const leftoverSessionNote = (computer: Computer, replaced = false): string | undefined => {
  const expiresAt = Date.parse(computer.token_expires_at);
  if (!(expiresAt > Date.now())) return undefined;
  const which = replaced ? "previous session" : "session";
  const until = new Date(expiresAt).toLocaleDateString();
  return `Nexul's ${which} on this computer stays valid until ${until}. Revoke it on the computer with \`t3 auth session list\`, then \`t3 auth session revoke <id>\`.`;
};

export const tunnelOnline = (status: TunnelStatus) => status.tunnel === "healthy" || status.tunnel === "degraded";
export const tunnelConnected = (status: TunnelStatus) => tunnelOnline(status) && status.harness_reachable;

// The pair-a-computer dialog's steps, in order.
export const PAIRING_STEPS = [
  { value: "connect", label: "Connect" },
  { value: "pair", label: "Pair T3 Code" },
  { value: "setup", label: "Set up" },
] as const;

export type PairingStep = (typeof PAIRING_STEPS)[number]["value"];

export type SetupTurnState = "running" | "confirmed" | "failed";

// Mirrors pairing.SetupTurnKind: a full setup of the provider, or a skills update that rewrote the shared skill folders.
export type SetupTurnKind = "setup" | "skills";

export interface SetupTurnSummary {
  run_id: string;
  turn_id: string;
  provider: string;
  provider_name: string;
  kind: SetupTurnKind;
  state: SetupTurnState;
  status: string;
  // The model slug the turn ran on; empty means the provider's own default.
  model?: string;
  updated_at: string;
}

// Keyed by the provider's driver kind; a null confirmed_at means unconfirmed.
export interface ProviderSetup {
  provider: string;
  confirmed_at: string | null;
  skills: string[];
  // The nexul-memory version last recorded for it; outdated is a signal to update skills, never a block.
  skills_version: string;
  skills_outdated: boolean;
}

// Read-only in the browser: confirmations are written only by an agent through MCP (ADR 0063).
interface ComputerSetupState {
  computer_id: string;
  confirmed_at: string | null;
  providers: ProviderSetup[];
  turns: SetupTurnSummary[];
}

// The Set up step's saved choices, by lower-case driver kind; ComputerSetup carries them and Done or a run saves them.
export interface SetupChoices {
  // Switched off; the step opens with them off.
  skipped_providers: string[];
  // A key present with "" is the provider's own default; a key absent falls back to the preselection.
  models: Record<string, string>;
  model_options: Record<string, OptionSetting[]>;
  folder: string;
}

export type ComputerSetup = ComputerSetupState & SetupChoices;

export interface SetupRun {
  run_id: string;
  computer_id: string;
  folder?: string;
  providers: { provider: string; name: string; model?: string }[];
}

// Queued: in the run just started, its turn not begun yet.
export interface SetupRunRow {
  provider: string;
  name: string;
  kind: SetupTurnKind;
  state: SetupTurnState | "queued";
  status: string;
  model: string;
  turnId?: string;
}

export const SETUP_STATE_LABEL: Record<SetupRunRow["state"], string> = {
  queued: "Waiting",
  running: "Running",
  confirmed: "Confirmed",
  failed: "Failed",
};

const turnRow = (t: SetupTurnSummary): SetupRunRow => ({
  provider: t.provider,
  name: t.provider_name || t.provider,
  kind: t.kind,
  state: t.state,
  status: t.status,
  model: t.model ?? "",
  turnId: t.turn_id,
});

// The run's providers in order, each at its turn in that run or queued, then any other provider's newest turn.
export const setupRunRows = (turns: SetupTurnSummary[], run: SetupRun | undefined): SetupRunRow[] => {
  const inRun = (run?.providers ?? []).map((p): SetupRunRow => {
    const turn = turns.find((t) => t.provider === p.provider && t.run_id === run?.run_id);
    if (turn) return turnRow(turn);
    return { provider: p.provider, name: p.name, kind: "setup", state: "queued", status: "Waiting for its turn", model: p.model ?? "" };
  });
  const rest = turns.filter((t) => !inRun.some((r) => r.provider === t.provider)).map(turnRow);
  return [...inRun, ...rest];
};

// One provider the Set up step picks a model for, keyed by driver kind like the setup turns; "" is the provider's own default.
export interface SetupModelChoice {
  provider: string;
  name: string;
  instance: HarnessProvider;
  preselected: string;
  preselectedOptions: OptionSetting[];
}

type Preselection = Pick<SetupModelChoice, "preselected" | "preselectedOptions">;

// The saved pick wins, then the defaults' model on the default provider, then the provider's own default.
const preselect = (p: HarnessProvider, defaults: PairingDefaults | undefined, saved: SetupChoices | undefined): Preselection => {
  const driver = p.driver.toLowerCase();
  const savedModel = saved?.models[driver];
  if (savedModel !== undefined) return { preselected: savedModel, preselectedOptions: saved?.model_options[driver] ?? [] };
  const fromDefaults = defaults?.provider === p.id ? p.models.find((m) => m.slug === defaults.model)?.slug : undefined;
  if (fromDefaults) return { preselected: fromDefaults, preselectedOptions: defaults?.model_options ?? [] };
  return { preselected: p.models.find((m) => m.is_default)?.slug ?? "", preselectedOptions: [] };
};

// The harness's first instance of each driver, as setup runs them.
export const setupModelChoices = (
  providers: HarnessProvider[],
  defaults: PairingDefaults | undefined,
  saved?: SetupChoices,
): SetupModelChoice[] =>
  providers.flatMap((p, i) => {
    if (providers.findIndex((o) => o.driver.toLowerCase() === p.driver.toLowerCase()) !== i) return [];
    return [{ provider: p.driver.toLowerCase(), name: p.name, instance: p, ...preselect(p, defaults, saved) }];
  });

// The row the transcript follows: the running one, else the last that ran so a finished run stays put, else the first.
export const followedSetupRow = (rows: SetupRunRow[]): SetupRunRow | undefined =>
  rows.find((r) => r.state === "running") ?? [...rows].reverse().find((r) => r.state !== "queued") ?? rows[0];

export const setupRunning = (rows: SetupRunRow[]) => rows.some((r) => r.state === "running" || r.state === "queued");

export interface ProviderSetupLine {
  provider: string;
  name: string;
  state: SetupTurnState | "unconfirmed";
  // The kind of its newest turn, so a running skills update reads as one.
  kind: SetupTurnKind;
  confirmedAt: string | null;
  skillsOutdated: boolean;
}

// One line per provider the computer has a confirmation row or a setup turn for; a running turn wins over the stored state.
export const providerSetupLines = (setup: ComputerSetup): ProviderSetupLine[] => {
  const providers = [...new Set([...setup.providers.map((p) => p.provider), ...setup.turns.map((t) => t.provider)])];
  return providers.map((provider) => {
    const turn = setup.turns.find((t) => t.provider === provider);
    const row = setup.providers.find((p) => p.provider === provider);
    const confirmedAt = row?.confirmed_at ?? null;
    const base = { provider, name: turn?.provider_name || provider, kind: turn?.kind ?? "setup", confirmedAt, skillsOutdated: row?.skills_outdated ?? false };
    if (turn?.state === "running") return { ...base, state: "running" };
    if (confirmedAt) return { ...base, state: "confirmed" };
    if (turn?.state === "failed") return { ...base, state: "failed" };
    return { ...base, state: "unconfirmed" };
  });
};

// The hint every update dot for out-of-date skills reads out.
export const SKILLS_OUTDATED = "skills out of date";

export const hasOutdatedSkills = (setup: ComputerSetup) => setup.providers.some((p) => p.skills_outdated);

// Update skills fits only a confirmed computer whose every provider line is confirmed and where skills are what is left.
export const onlySkillsOutdated = (setup: ComputerSetup) => {
  const lines = providerSetupLines(setup);
  return setup.confirmed_at !== null && lines.every((l) => l.state === "confirmed") && lines.some((l) => l.skillsOutdated);
};

// Mirrors pairing.RefusalDetails, the error envelope's details when agent work is refused at target resolution.
export interface RefusalDetails {
  reason: string;
  computer_id?: string;
  computer?: string;
  provider_id?: string;
  provider?: string;
}

// Mirrors pairing.ReasonSetupRequired, the refusal whose fix is running setup on the computer it names.
export const SETUP_REQUIRED_REASON = "setup_required";

// Your settings → T3 pairing with the computer's Set up step open.
export const computerSetupPath = (computerId: string) =>
  `/settings/pairing?setup=${encodeURIComponent(computerId)}`;

// The computer a setup refusal names, read from the error envelope's details; undefined for any other error.
export const setupRefusalComputerId = (error: unknown): string | undefined => {
  const details = (error as AxiosError<ApiErrorBody> | undefined)?.response?.data?.details as RefusalDetails | undefined;
  if (details?.reason !== SETUP_REQUIRED_REASON) return undefined;
  return details.computer_id;
};

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

// A user's pairing defaults, for chats outside a project and projects they haven't linked.
export interface PairingDefaults {
  default_computer_id?: string;
  fallback_project_id?: string;
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
}

// The caller's own link for one project (ADR 0102); an empty computer_id means they never set one.
export interface ProjectLink {
  project_id?: string;
  computer_id?: string;
  harness_project_id?: string;
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
}

export interface HarnessProject {
  id: string;
  title: string;
  path: string;
}

export interface HarnessProviderModel {
  slug: string;
  name: string;
  is_default?: boolean;
  // Where a routed model comes from, such as "GitHub Copilot" under OpenCode.
  sub_provider?: string;
  is_new?: boolean;
  is_legacy?: boolean;
  options?: ModelOption[];
}

export interface OptionChoice {
  id: string;
  label: string;
  description?: string;
  is_default?: boolean;
}

// One setting the harness offers per model (reasoning level, context window, fast mode); a switch has no choices.
export interface ModelOption {
  id: string;
  label: string;
  description?: string;
  type: "select" | "boolean";
  choices?: OptionChoice[];
  default_on?: boolean;
}

// One option set for a turn; an option left out runs on the harness default.
export interface OptionSetting {
  id: string;
  value: string | boolean;
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
  model_options: z.array(z.object({ id: z.string(), value: z.union([z.string(), z.boolean()]) })),
});

export type PairingDefaultsFormData = z.infer<typeof PairingDefaultsFormSchema>;

export const ProjectLinkFormSchema = z.object({
  computer_id: z.string().trim().min(1, "Pick a computer"),
  harness_project_id: z.string().trim().min(1, "T3 project id is required"),
  provider: z.string().trim(),
  model: z.string().trim(),
  model_options: z.array(z.object({ id: z.string(), value: z.union([z.string(), z.boolean()]) })),
});

export type ProjectLinkFormData = z.infer<typeof ProjectLinkFormSchema>;

// The settings UI's threshold for flagging a computer as about to act like unpaired.
export const EXPIRY_WARNING_DAYS = 5;

// Whether the caller can run a play right now: the resolved computer, provider, and model, or a reason it
// can't, joined against presence. provider/model travel here so the run dialog's pill can preselect them
// without a second call to the same resolve endpoint.
export type HarnessReadiness =
  | { state: "ready"; computerId: string; provider: string; model: string; modelOptions: OptionSetting[] }
  | { state: "unpaired" | "expired" | "no_harness_project" | "no_default_computer" | "offline"; message: string };

// Copy for every non-ready state; shared by the hook's join and the settings readiness line. Says harness, never computer.
export const HARNESS_READINESS_COPY: Record<Exclude<HarnessReadiness["state"], "ready">, string> = {
  unpaired: "Pair a harness in Settings to run plays",
  expired: "Your harness pairing has expired, re-pair it in Settings",
  no_harness_project: "Link this project in Settings → T3 pairing → Projects, or set a fallback under Defaults",
  no_default_computer: "Several harnesses are paired, pick a default in Settings",
  offline: "Your harness is offline",
};
