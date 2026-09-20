import type { AutomationsApi, RemoteState } from "./automations-api.ts";
import type { AutomationTarget } from "./host-tokens.ts";
import { log } from "./log.ts";
import type { WorkerFactory, WorkerHandle } from "./worker.ts";

export interface SupervisorConfig {
  serverUrl: string;
  timeoutMs: number;
  heartbeatTimeoutMs: number;
  memoryMb: number;
}

interface TrackedState {
  activeVersionId: string | null;
  updatedAt: string;
  running: boolean;
}

// Supervisor is the host's whole job: for each known automation, ask
// the server what should be running and reconcile a worker to match. One
// poll tick, one reconcile pass — no queue, no scheduler, deliberately.
export class Supervisor {
  private readonly tracked = new Map<string, TrackedState>();
  private readonly handles = new Map<string, WorkerHandle>();

  constructor(
    private readonly targets: AutomationTarget[],
    private readonly api: AutomationsApi,
    private readonly workers: WorkerFactory,
    private readonly cfg: SupervisorConfig,
  ) {}

  async pollOnce(): Promise<void> {
    for (const target of this.targets) {
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
      this.tracked.set(target.id, toTracked(state, false));
      return;
    }

    if (!state.activeCode) {
      log("warn", "automation enabled but has no active version yet", { automationId: target.id });
      return;
    }

    const changed = !prev || prev.activeVersionId !== state.activeVersionId || prev.updatedAt !== state.updatedAt;
    if (prev?.running && !changed) return;

    if (prev?.running) await this.stop(target.id);
    this.spawn(target, state);
  }

  private spawn(target: AutomationTarget, state: RemoteState): void {
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
      },
      () => this.markStale(target.id),
    );
    this.handles.set(target.id, handle);
    this.tracked.set(target.id, toTracked(state, true));
    log("info", "automation worker started", { automationId: target.id, name: target.name, versionId: state.activeVersionId });
  }

  private async stop(id: string): Promise<void> {
    const handle = this.handles.get(id);
    this.handles.delete(id);
    if (handle) await handle.terminate();
  }

  private markStale(id: string): void {
    const prev = this.tracked.get(id);
    if (prev) this.tracked.set(id, { ...prev, running: false });
    this.handles.delete(id);
  }
}

function toTracked(state: RemoteState, running: boolean): TrackedState {
  return { activeVersionId: state.activeVersionId, updatedAt: state.updatedAt, running };
}
