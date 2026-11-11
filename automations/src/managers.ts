import type { AutomationsApi, ManagerReport } from "./automations-api.ts";
import { log } from "./log.ts";
import type { Resolver } from "./resolver.ts";

export type OpenResolver = (token: string) => Resolver;

// The secrets managers this host holds a token for, by workspace. Each token's client is opened once at start and kept;
// the supervisor alone holds them, and a worker thread never sees one (ADR 0182).
export class WorkspaceManagers {
  private readonly resolvers = new Map<string, Resolver>();
  private reports: ManagerReport[] = [];
  private reported = false;

  constructor(
    private readonly tokens: Map<string, string>,
    private readonly open: OpenResolver,
  ) {}

  // Opens one client per token and records each one's state; a failed token is a state, never a thrown error.
  async start(): Promise<void> {
    const entries = [...this.tokens];
    this.reports = await Promise.all(
      entries.map(async ([workspace, token]) => {
        const resolver = this.open(token);
        this.resolvers.set(workspace, resolver);
        const state = await resolver.check();
        log("info", "secrets manager checked", { workspace, manager: resolver.scheme, state });
        return { workspace, manager: resolver.scheme, state };
      }),
    );
  }

  resolverFor(workspace: string): Resolver | undefined {
    return this.resolvers.get(workspace);
  }

  // Tells the server the states once it answers, empty included so a removed token clears; a failed report is tried
  // again at the next poll.
  async reportOnce(api: AutomationsApi, serverUrl: string, credential: string): Promise<void> {
    if (this.reported) return;
    try {
      await api.reportManagers(serverUrl, credential, this.reports);
      this.reported = true;
    } catch (err) {
      log("warn", "failed to report secrets managers", { error: err instanceof Error ? err.message : String(err) });
    }
  }
}
