import type { AutomationTarget, AutomationsApi, RemoteState } from "./automations-api.ts";
import { log } from "./log.ts";
import type { PoolResolver } from "./pool-resolution.ts";
import { realClock, type Clock, type SecretReader } from "./secret-check.ts";
import { Fingerprinter } from "./secret-fingerprints.ts";
import type { WorkerFactory, WorkerHandle } from "./worker.ts";

export interface SupervisorConfig {
  serverUrl: string;
  timeoutMs: number;
  heartbeatTimeoutMs: number;
  memoryMb: number;
}

interface TrackedState {
  activeVersionId: string | null;
  configValues: string;
  token: string;
  running: boolean;
}

// A run in flight is given this long to end before a changed secret restarts its automation anyway; the cut-off run is
// delivered again (ADR 0046).
const MAX_IDLE_WAIT_MS = 5 * 60_000;

// What one worker was handed at its last connection: a keyed hash per reference, never the value.
interface Held {
  workspaceId: string;
  refs: Map<string, string>;
}

export interface SupervisorDeps {
  fingerprinter: Fingerprinter;
  clock: Clock;
}

// Supervisor is the host's whole job: for each automation placed on this
// host, ask the server what should be running and reconcile a worker to
// match. One poll tick, one reconcile pass — no queue, no scheduler.
export class Supervisor {
  private readonly tracked = new Map<string, TrackedState>();
  private readonly handles = new Map<string, WorkerHandle>();
  private readonly held = new Map<string, Held>();
  private readonly specs = new Map<string, { target: AutomationTarget; state: RemoteState }>();
  private readonly busy = new Set<string>();
  private readonly waiting = new Map<string, () => void>();

  constructor(
    private readonly api: AutomationsApi,
    private readonly workers: WorkerFactory,
    private readonly cfg: SupervisorConfig,
    private readonly pool: PoolResolver,
    private readonly deps: SupervisorDeps = { fingerprinter: new Fingerprinter(), clock: realClock },
  ) {}

  // reconcile stops the workers of automations no longer placed here, then brings each placed one up to date.
  async reconcile(targets: AutomationTarget[]): Promise<void> {
    const placed = new Set(targets.map((t) => t.id));
    for (const id of [...this.tracked.keys()]) {
      if (placed.has(id)) continue;
      await this.stop(id);
      this.tracked.delete(id);
      log("info", "automation worker stopped: no longer placed on this host", { automationId: id });
    }
    for (const target of targets) {
      await this.reconcileOne(target);
    }
  }

  async stopAll(): Promise<void> {
    await Promise.all([...this.handles.keys()].map((id) => this.stop(id)));
  }

  private async reconcileOne(target: AutomationTarget): Promise<void> {
    let state: RemoteState;
    try {
      state = await this.api.fetchState(target, this.cfg.serverUrl);
    } catch (err) {
      log("error", "failed to fetch automation state", { automationId: target.id, error: err instanceof Error ? err.message : String(err) });
      return;
    }

    const prev = this.tracked.get(target.id);

    if (!state.enabled) {
      if (prev?.running) await this.stop(target.id);
      this.tracked.set(target.id, toTracked(state, target, false));
      return;
    }

    if (!state.activeCode) {
      log("warn", "automation enabled but has no active version yet", { automationId: target.id });
      return;
    }

    // Only what the worker holds restarts it: its own announce moves updated_at, so that would restart every new worker.
    const changed =
      !prev || prev.activeVersionId !== state.activeVersionId || prev.configValues !== state.configValues || prev.token !== target.token;
    if (prev?.running && !changed) return;

    if (prev?.running) await this.stop(target.id);
    this.spawn(target, state);
  }

  // Re-reads every held reference, one read per workspace, and restarts each automation whose handed-in value is no
  // longer the current one. A reference that does not resolve now counts as unchanged. Logs only ids and counts.
  async checkSecrets(reader: SecretReader): Promise<void> {
    const refsByWorkspace = new Map<string, Set<string>>();
    for (const held of this.held.values()) {
      const refs = refsByWorkspace.get(held.workspaceId) ?? new Set<string>();
      for (const ref of held.refs.keys()) refs.add(ref);
      refsByWorkspace.set(held.workspaceId, refs);
    }
    const checked = [...this.held].map(([id, held]) => ({ id, held }));
    const results = new Map<string, Awaited<ReturnType<SecretReader["resolve"]>>>();
    await Promise.all([...refsByWorkspace].map(async ([workspaceId, refs]) => void results.set(workspaceId, await reader.resolve(workspaceId, [...refs]))));
    for (const { id, held } of checked) {
      if (this.held.get(id) !== held) continue;
      const now = results.get(held.workspaceId);
      let changed = 0;
      for (const [ref, fingerprint] of held.refs) {
        const result = now?.get(ref);
        if (result?.ok && this.deps.fingerprinter.of(result.value) !== fingerprint) changed++;
      }
      if (changed > 0) this.restartWhenIdle(id, changed);
    }
  }

  private restartWhenIdle(id: string, changed: number): void {
    if (this.waiting.has(id)) return;
    log("info", "automation holds changed secret values; restarting it once idle", { automationId: id, changed });
    if (!this.busy.has(id)) {
      void this.restart(id);
      return;
    }
    this.waiting.set(id, this.deps.clock.after(MAX_IDLE_WAIT_MS, () => void this.restart(id)));
  }

  // The thread reconnects with the spec it last ran and resolves the pool afresh.
  private async restart(id: string): Promise<void> {
    const spec = this.specs.get(id);
    if (!spec) return;
    await this.stop(id);
    this.spawn(spec.target, spec.state);
  }

  private runState(id: string, held: Held, running: boolean): void {
    if (this.held.get(id) !== held) return;
    if (running) {
      this.busy.add(id);
      return;
    }
    this.busy.delete(id);
    if (this.waiting.has(id)) void this.restart(id);
  }

  private remember(id: string, held: Held, entries: unknown, values: Record<string, string>): void {
    if (this.held.get(id) !== held) return;
    const asked = typeof entries === "object" && entries !== null ? (entries as Record<string, unknown>) : {};
    held.refs.clear();
    for (const [name, value] of Object.entries(values)) {
      const ref = asked[name];
      if (typeof ref === "string") held.refs.set(ref, this.deps.fingerprinter.of(value));
    }
  }

  private forget(id: string): void {
    this.waiting.get(id)?.();
    this.waiting.delete(id);
    this.held.delete(id);
    this.specs.delete(id);
    this.busy.delete(id);
  }

  private spawn(target: AutomationTarget, state: RemoteState): void {
    const held: Held = { workspaceId: target.workspace_id, refs: new Map() };
    this.held.set(target.id, held);
    this.specs.set(target.id, { target, state });
    const handle = this.workers.spawn(
      {
        automationId: target.id,
        automationName: target.name,
        code: state.activeCode as string,
        url: this.cfg.serverUrl,
        token: target.token,
        timeoutMs: this.cfg.timeoutMs,
        heartbeatTimeoutMs: this.cfg.heartbeatTimeoutMs,
        memoryMb: this.cfg.memoryMb,
        resolvePool: async (entries) => {
          const values = await this.pool.resolve({ workspaceId: target.workspace_id, automationId: target.id, automationName: target.name }, entries);
          this.remember(target.id, held, entries, values);
          return values;
        },
        onRunState: (running) => this.runState(target.id, held, running),
      },
      () => this.markStale(target.id),
    );
    this.handles.set(target.id, handle);
    this.tracked.set(target.id, toTracked(state, target, true));
    log("info", "automation worker started", { automationId: target.id, automationName: target.name, versionId: state.activeVersionId });
  }

  private async stop(id: string): Promise<void> {
    const handle = this.handles.get(id);
    this.handles.delete(id);
    this.forget(id);
    if (handle) await handle.terminate();
  }

  private markStale(id: string): void {
    const prev = this.tracked.get(id);
    if (prev) this.tracked.set(id, { ...prev, running: false });
    this.handles.delete(id);
    this.forget(id);
  }
}

function toTracked(state: RemoteState, target: AutomationTarget, running: boolean): TrackedState {
  return { activeVersionId: state.activeVersionId, configValues: state.configValues, token: target.token, running };
}
