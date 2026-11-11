import type { AutomationsApi, ManagerReport } from "./automations-api.ts";
import { log } from "./log.ts";
import type { ManagerState, Resolver } from "./resolver.ts";

export type OpenResolver = (token: string) => Resolver;

const CHECK_TIMEOUT_MS = 30_000;
const RETRY_BASE_MS = 5_000;
const RETRY_MAX_MS = 5 * 60_000;

export interface ManagerTimers {
  delay: (ms: number) => Promise<void>;
  random: () => number;
}

const realTimers: ManagerTimers = {
  delay: (ms) => new Promise((resolve) => void setTimeout(resolve, ms).unref()),
  random: Math.random,
};

// The secrets managers this host holds a token for, by workspace. Each token's client is opened in the background at
// start and kept; an unreachable one is tried again with backoff, a rejected token is final. The supervisor alone holds
// the tokens, and a worker thread never sees one (ADR 0182).
export class WorkspaceManagers {
  private readonly resolvers = new Map<string, Resolver>();
  private readonly states = new Map<string, ManagerState>();
  private lastReported: string | undefined;

  constructor(
    private readonly tokens: Map<string, string>,
    private readonly open: OpenResolver,
    private readonly timers: ManagerTimers = realTimers,
  ) {}

  // Starts one check per token without waiting for any: polling must not wait on a manager that hangs. A failed token
  // is a state, never a thrown error.
  start(): void {
    for (const [workspace, token] of this.tokens) {
      const resolver = this.open(token);
      this.resolvers.set(workspace, resolver);
      void this.settle(workspace, resolver);
    }
  }

  private async settle(workspace: string, resolver: Resolver): Promise<void> {
    for (let attempt = 0; ; attempt++) {
      const state = await this.check(resolver);
      if (this.states.get(workspace) !== state) log("info", "secrets manager checked", { workspace, manager: resolver.scheme, state });
      this.states.set(workspace, state);
      if (state !== "unreachable") return;
      await this.timers.delay(this.backoff(attempt));
    }
  }

  // A check that outlasts its time limit answers unreachable; its late answer is dropped.
  private check(resolver: Resolver): Promise<ManagerState> {
    const timeout = this.timers.delay(CHECK_TIMEOUT_MS).then((): ManagerState => "unreachable");
    return Promise.race([resolver.check(), timeout]);
  }

  private backoff(attempt: number): number {
    const base = Math.min(RETRY_BASE_MS * 2 ** attempt, RETRY_MAX_MS);
    return base + this.timers.random() * (base / 2);
  }

  resolverFor(workspace: string): Resolver | undefined {
    return this.resolvers.get(workspace);
  }

  // Tells the server the states once every token's first outcome is known, and again whenever they change; empty
  // included so a removed token clears. A failed report is tried again at the next poll.
  async report(api: AutomationsApi, serverUrl: string, credential: string): Promise<void> {
    if (this.states.size < this.tokens.size) return;
    const reports: ManagerReport[] = [...this.resolvers].map(([workspace, resolver]) => ({
      workspace,
      manager: resolver.scheme,
      state: this.states.get(workspace) ?? "unreachable",
    }));
    const snapshot = JSON.stringify(reports);
    if (snapshot === this.lastReported) return;
    try {
      await api.reportManagers(serverUrl, credential, reports);
      this.lastReported = snapshot;
    } catch (err) {
      log("warn", "failed to report secrets managers", { error: err instanceof Error ? err.message : String(err) });
    }
  }
}
