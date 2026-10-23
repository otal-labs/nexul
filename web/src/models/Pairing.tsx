import type { AxiosError } from "axios";
import { z } from "zod";

import type { ApiErrorBody } from "@/api/client";
import type { ComputerFacts } from "@/models/ComputerFacts";
import type { SharedComputer } from "@/models/SharedComputer";

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
  // A tunnel left from pairing through one, until the server's cleanup deletes it.
  tunnel?: ComputerTunnel;
  // Absent until the computer's runner first reports; facts_at is when they last changed.
  facts?: ComputerFacts;
  facts_at?: string;
  // The personal runner that reaches it; absent before one enrolls and after one is revoked.
  runner?: ComputerRunner;
  // Why the last pairing through the runner failed; absent once one succeeds.
  pair_error?: string;
  // Set on another person's computer they share with the viewer, which then carries only its name and runner.connected.
  shared?: SharedComputer;
  // Its place in its owner's order, first runs first (ADR 0165).
  position: number;
  // How many play runs it takes at a time, and how many are going on it now (ADR 0166).
  runs_at_once: number;
  running: number;
  // The T3 project an @Agent chat outside any project runs in here; "" when none is picked.
  chat_project_id: string;
}

// Mirrors pairing.PassOver: one computer a run could not land on, with the line every surface shows as is.
export interface PassOver {
  computer_id: string;
  name: string;
  why: "offline" | "expired_token" | "setup_required" | "no_model" | "no_checkout" | "ambiguous_checkout" | "no_chat_folder" | "full" | "clone_failed";
  text: string;
}

// "laptop is offline; build-box is full (2 of 2 running)".
export const passOverLines = (passes: PassOver[]) => passes.map((p) => p.text).join("; ");

export const MIN_RUNS_AT_ONCE = 1;
export const MAX_RUNS_AT_ONCE = 5;

export interface ComputerRunner {
  connected: boolean;
  last_seen: string;
  // Dropped moments ago and expected back; it reads as offline once the server's grace ends.
  reconnecting: boolean;
}

// A computer waiting for its runner, with the one-time command that installs it; Windows stays empty until it installs there.
export interface ComputerEnrollment {
  computer: Computer;
  token: string;
  expires_at: string;
  commands: { unix: string; windows: string };
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

// Go's zero time: a computer holds no session until T3 Code pairs through its runner.
export const stillPairing = (computer: Computer) => !(Date.parse(computer.token_expires_at) > 0);

// T3 Code lets no client end its own session, so a removed or replaced pairing's session lives on until it expires.
export const leftoverSessionNote = (computer: Computer, replaced = false): string | undefined => {
  const expiresAt = Date.parse(computer.token_expires_at);
  if (!(expiresAt > Date.now())) return undefined;
  const which = replaced ? "previous session" : "session";
  const until = new Date(expiresAt).toLocaleDateString();
  return `Nexul's ${which} on this computer stays valid until ${until}. Revoke it on the computer with \`t3 auth session list\`, then \`t3 auth session revoke <id>\`.`;
};

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
    return { provider: p.provider, name: p.name, kind: "setup", state: "queued", status: "Starting", model: p.model ?? "" };
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
    // Setup offers every model of every provider, set up or not.
    const instance = { ...p, needs_setup: false, skipped: false };
    return [{ provider: p.driver.toLowerCase(), name: p.name, instance, ...preselect(p, defaults, saved) }];
  });

// The row that opens by itself: the running one, else the first failure; a settled run stays folded.
export const openSetupRow = (rows: SetupRunRow[]): SetupRunRow | undefined =>
  rows.find((r) => r.state === "running") ?? rows.find((r) => r.state === "failed");

export const setupRunning = (rows: SetupRunRow[]) => rows.some((r) => r.state === "running" || r.state === "queued");

// One provider in the Set up step: its switch, its model, and where its setup stands; unset has never been confirmed.
export interface SetupProviderRow {
  provider: string;
  name: string;
  driver: string;
  on: boolean;
  state: SetupRunRow["state"] | "unset";
  // Absent when T3 Code no longer lists the provider: its last turn still shows, with nothing to pick or run.
  choice?: SetupModelChoice;
  run?: SetupRunRow;
}

// Every provider T3 Code lists, in its order, then any that only a past turn remembers.
export const setupProviderRows = (choices: SetupModelChoice[], runRows: SetupRunRow[], setup: ComputerSetup): SetupProviderRow[] => {
  const confirmed = (provider: string) => setup.providers.some((p) => p.provider === provider && p.confirmed_at);
  const row = (provider: string, name: string, driver: string, choice?: SetupModelChoice): SetupProviderRow => {
    const run = runRows.find((r) => r.provider === provider);
    const state = run?.state ?? (confirmed(provider) ? "confirmed" : "unset");
    return { provider, name, driver, on: !setup.skipped_providers.includes(provider), state, ...(choice && { choice }), ...(run && { run }) };
  };
  const listed = choices.map((c) => row(c.provider, c.name, c.instance.driver, c));
  const gone = runRows.filter((r) => !choices.some((c) => c.provider === r.provider)).map((r) => row(r.provider, r.name, r.provider));
  return [...listed, ...gone];
};

// The muted line under a provider's name.
export const setupProviderLine = (row: SetupProviderRow) => {
  if (!row.on) return "Off · plays leave it out";
  if (row.state === "unset") return "Not set up yet";
  if (row.state === "queued") return "Starting";
  if (row.state === "confirmed") return row.run?.kind === "skills" ? "Confirmed · skills updated" : "Confirmed";
  return row.run?.status || SETUP_STATE_LABEL[row.state];
};

// Waiting on the person: switched on and either never set up or failed.
export const needsSetup = (row: SetupProviderRow) => row.on && (row.state === "unset" || row.state === "failed");

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

export type SetupSummaryTone = "success" | "warning" | "destructive" | "info";

export interface SetupSummary {
  state: "running" | "failed" | "missing" | "outdated" | "confirmed";
  tone: SetupSummaryTone;
  text: string;
  // The providers the status is about, shown as their marks after it.
  providers?: ProviderSetupLine[];
}

// A computer's setup in one status: what is running, then what failed, then what is missing, then stale skills.
export const setupSummary = (setup: ComputerSetup): SetupSummary => {
  const lines = providerSetupLines(setup);
  const only = (state: ProviderSetupLine["state"]) => lines.filter((l) => l.state === state);
  const running = lines.find((l) => l.state === "running");
  if (running) return { state: "running", tone: "warning", text: running.kind === "skills" ? "Updating skills" : `Setting up ${running.name}` };
  if (lines.some((l) => l.state === "failed")) return { state: "failed", tone: "destructive", text: "Setup failed", providers: only("failed") };
  if (setup.confirmed_at === null) return { state: "missing", tone: "warning", text: "Needs setup" };
  if (lines.some((l) => l.skillsOutdated)) return { state: "outdated", tone: "info", text: "Skills out of date" };
  return { state: "confirmed", tone: "success", text: "Setup confirmed", providers: only("confirmed") };
};

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
  passed_over?: PassOver[];
}

// Mirrors pairing.ReasonSetupRequired, the refusal whose fix is running setup on the computer it names.
export const SETUP_REQUIRED_REASON = "setup_required";

// Your settings → Computers with the computer's Set up step open.
export const computerSetupPath = (computerId: string) =>
  `/settings/pairing?setup=${encodeURIComponent(computerId)}`;

// Mirrors pairing.ReasonNeedsLocation: no computer of the person's has the project, so their run asks where (ADR 0165).
export const NEEDS_LOCATION_REASON = "needs_location";

export const isNeedsLocationRefusal = (error: unknown) =>
  ((error as AxiosError<ApiErrorBody> | undefined)?.response?.data?.details as RefusalDetails | undefined)?.reason === NEEDS_LOCATION_REASON;

// The computer a setup refusal names, read from the error envelope's details; undefined for any other error.
export const setupRefusalComputerId = (error: unknown): string | undefined => {
  const details = (error as AxiosError<ApiErrorBody> | undefined)?.response?.data?.details as RefusalDetails | undefined;
  if (details?.reason !== SETUP_REQUIRED_REASON) return undefined;
  return details.computer_id;
};

// Labels for the harness kinds a computer can be paired with; keys match the Go harness.Kind values.
export const HARNESS_LABELS: Record<string, string> = { t3code: "T3 Code", "t3code-v2": "T3 Code" };
export const harnessLabel = (kind: string) => HARNESS_LABELS[kind] ?? kind;

// Where a new T3 thread starts: the T3 project's folder, or a fresh git worktree so parallel runs never share files.
export const StartIn = { Folder: "folder", Worktree: "worktree" } as const;
export type StartIn = (typeof StartIn)[keyof typeof StartIn];

export const START_IN_OPTIONS = [
  { value: StartIn.Folder, label: "Project folder" },
  { value: StartIn.Worktree, label: "New worktree per thread" },
];

export const START_IN_SHORT_LABELS: Record<StartIn, string> = { [StartIn.Folder]: "Project folder", [StartIn.Worktree]: "New worktree" };

// How a run picks among the person's ready computers: the first with room, or the least busy for its size (ADR 0165).
export const Placement = { InOrder: "in_order", Spread: "spread" } as const;
export type Placement = (typeof Placement)[keyof typeof Placement];

export const PLACEMENT_OPTIONS = [
  { value: Placement.InOrder, label: "First in order" },
  { value: Placement.Spread, label: "Least busy" },
];

// What a person's runs use wherever a project link says nothing.
export interface PairingDefaults {
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
  start_in?: StartIn;
  placement: Placement;
  // Clones a project's repository onto a computer without a checkout of it, before a run there.
  clone_automatically: boolean;
}

// The T3 project the person picked for a project on one computer.
export interface Checkout {
  computer_id: string;
  harness_project_id: string;
}

// The caller's own settings for one project (ADR 0102, ADR 0165); empty computers follows their order.
export interface ProjectLink {
  project_id: string;
  provider?: string;
  model?: string;
  model_options?: OptionSetting[];
  // Unset falls through to the person's defaults.
  start_in?: StartIn;
  // Only with its own computers; unset follows the person's.
  placement?: Placement;
  computers: string[];
  checkouts: Checkout[];
}

// A project link that changes anything a run there would otherwise take from the person's defaults and order.
export const linkSaysSomething = (link: ProjectLink | undefined) =>
  !!link && (link.computers.length > 0 || !!link.provider || !!link.model || !!link.start_in);

export interface HarnessProject {
  id: string;
  title: string;
  path: string;
  // The checkout's git remote as T3 normalizes it, such as github.com/acme/api; absent until T3 has resolved it.
  repository?: string;
}

// can_clone is false for a computer paired before Nexul could clone there.
export interface HarnessProjects {
  projects: HarnessProject[];
  can_clone: boolean;
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
  // Shown but not runnable until setup confirms it.
  needs_setup: boolean;
  // Switched off in the Set up step: pickers leave it out.
  skipped: boolean;
}

export const RenameComputerFormSchema = z.object({ name: z.string().trim().min(1, "Name this computer") });

export type RenameComputerFormData = z.infer<typeof RenameComputerFormSchema>;

// The inputs the pairing routes key a failure under, so it shows on the field that caused it.
export const PAIR_FIELDS = ["name"] as const satisfies readonly (keyof RenameComputerFormData)[];
export type PairField = (typeof PAIR_FIELDS)[number];

export const PairingDefaultsFormSchema = z.object({
  provider: z.string(),
  model: z.string(),
  model_options: z.array(z.object({ id: z.string(), value: z.union([z.string(), z.boolean()]) })),
  start_in: z.enum([StartIn.Folder, StartIn.Worktree]),
  placement: z.enum([Placement.InOrder, Placement.Spread]),
  clone_automatically: z.boolean(),
});

export type PairingDefaultsFormData = z.infer<typeof PairingDefaultsFormSchema>;

export const ProjectLinkFormSchema = z.object({
  provider: z.string().trim(),
  model: z.string().trim(),
  model_options: z.array(z.object({ id: z.string(), value: z.union([z.string(), z.boolean()]) })),
  start_in: z.enum(["", StartIn.Folder, StartIn.Worktree]),
  placement: z.enum(["", Placement.InOrder, Placement.Spread]),
  computers: z.array(z.string()),
  checkouts: z.array(z.object({ computer_id: z.string(), harness_project_id: z.string() })),
});

export type ProjectLinkFormData = z.infer<typeof ProjectLinkFormSchema>;

// Whether the caller can run a play right now: the computer it would land on, with its checkout, provider, model and the
// computers passed over before it, or a reason it can't. provider/model travel here so the run dialog's pill can preselect
// them without a second call to the same resolve endpoint.
export type HarnessReadiness =
  | {
      state: "ready";
      computerId: string;
      harnessProjectId: string;
      provider: string;
      model: string;
      modelOptions: OptionSetting[];
      passedOver: PassOver[];
    }
  | {
      state: "unpaired" | "expired" | "needs_location" | "no_computer_ready" | "offline";
      message: string;
      link?: ReadinessLink;
      passedOver?: PassOver[];
    };

// A run asks where instead of refusing when no computer of the person's has the project (ADR 0165).
export const canChooseRunLocation = (readiness: HarnessReadiness): boolean => readiness.state === "needs_location";

// Where a blocked state is fixed, kept beside its sentence so the sentence still reads alone as a tooltip.
export interface ReadinessLink {
  to: string;
  label: string;
}

export const COMPUTERS_SETTINGS_PATH = "/settings/pairing";
export const ADD_COMPUTER_LINK: ReadinessLink = { to: COMPUTERS_SETTINGS_PATH, label: "Add a computer" };

// Copy for every non-ready state; shared by the hook's join and the settings readiness line.
export const HARNESS_READINESS_COPY: Record<Exclude<HarnessReadiness["state"], "ready">, { message: string; link?: ReadinessLink }> = {
  unpaired: { message: "No computer is paired to run plays.", link: ADD_COMPUTER_LINK },
  expired: {
    message: "Your computer's pairing has expired or was revoked.",
    link: { to: COMPUTERS_SETTINGS_PATH, label: "Re-pair it" },
  },
  needs_location: {
    message: "None of your computers has this project yet.",
    link: { to: `${COMPUTERS_SETTINGS_PATH}/projects`, label: "Add it to a computer" },
  },
  no_computer_ready: {
    message: "None of your computers can take a run right now.",
    link: { to: COMPUTERS_SETTINGS_PATH, label: "Your computers" },
  },
  offline: { message: "T3 Code on your computer is offline." },
};
