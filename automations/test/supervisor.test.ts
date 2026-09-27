import { describe, expect, test } from "bun:test";
import type { AutomationTarget, AutomationsApi, RemoteState } from "../src/automations-api.ts";
import { Supervisor } from "../src/supervisor.ts";
import type { SpawnOptions, WorkerFactory, WorkerHandle } from "../src/worker.ts";

const target: AutomationTarget = { id: "a1", name: "Ticket finished", token: "dat_h.a1.x" };
const other: AutomationTarget = { id: "a2", name: "PR opened", token: "dat_h.a2.y" };
const cfg = { serverUrl: "http://server", timeoutMs: 30_000, heartbeatTimeoutMs: 90_000, memoryMb: 128 };

function fakeApi(states: RemoteState[]): AutomationsApi {
  let i = 0;
  return {
    fetchAssignments: async () => ({ removed: false, automations: [] }),
    fetchState: async () => states[Math.min(i++, states.length - 1)] as RemoteState,
  };
}

function fakeWorkers(): WorkerFactory & { spawns: SpawnOptions[]; terminated: string[] } {
  const spawns: SpawnOptions[] = [];
  const terminated: string[] = [];
  return {
    spawns,
    terminated,
    spawn(opts: SpawnOptions): WorkerHandle {
      spawns.push(opts);
      return {
        async terminate() {
          terminated.push(opts.automationId);
        },
      };
    },
  };
}

function state(overrides: Partial<RemoteState> = {}): RemoteState {
  return { enabled: true, updatedAt: "2026-01-01T00:00:00Z", activeVersionId: "v1", activeCode: "export default {}", ...overrides };
}

describe("Supervisor", () => {
  test("a failed fetch leaves the previous worker running untouched", async () => {
    const workers = fakeWorkers();
    let call = 0;
    const api: AutomationsApi = {
      fetchAssignments: async () => ({ removed: false, automations: [] }),
      fetchState: async () => {
        call += 1;
        if (call === 2) throw new Error("network blip");
        return state();
      },
    };
    const sup = new Supervisor(api, workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(1);
    expect(workers.terminated).toHaveLength(0);
  });

  test("skips spawning when enabled but there is no active version yet", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state({ activeCode: null, activeVersionId: null })]), workers, cfg);

    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(0);
  });

  test("spawns a worker with the automation's host-scoped token", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state()]), workers, cfg);

    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(1);
    expect(workers.spawns[0]?.automationId).toBe("a1");
    expect(workers.spawns[0]?.token).toBe("dat_h.a1.x");
  });

  test("does not respawn when nothing changed", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state(), state()]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(1);
  });

  test("respawns when the active version changes", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state({ activeVersionId: "v1" }), state({ activeVersionId: "v2" })]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(2);
    expect(workers.terminated).toEqual(["a1"]);
  });

  test("respawns when config/enabled state's updated_at changes", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state({ updatedAt: "t1" }), state({ updatedAt: "t2" })]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(2);
  });

  test("respawns with the new token when the instance hands out another", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state(), state()]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([{ ...target, token: "dat_h.a1.rotated" }]);

    expect(workers.terminated).toEqual(["a1"]);
    expect(workers.spawns.map((s) => s.token)).toEqual(["dat_h.a1.x", "dat_h.a1.rotated"]);
  });

  test("terminates the worker when the automation is disabled", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state({ enabled: true }), state({ enabled: false })]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([target]);

    expect(workers.spawns).toHaveLength(1);
    expect(workers.terminated).toEqual(["a1"]);
  });

  test("stops the worker of an automation moved off this host and starts one placed on it", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state()]), workers, cfg);

    await sup.reconcile([target]);
    await sup.reconcile([other]);

    expect(workers.terminated).toEqual(["a1"]);
    expect(workers.spawns.map((s) => s.automationId)).toEqual(["a1", "a2"]);

    await sup.reconcile([other, target]);
    expect(workers.spawns.map((s) => s.automationId)).toEqual(["a1", "a2", "a1"]);
  });

  test("stopAll terminates every running worker", async () => {
    const workers = fakeWorkers();
    const sup = new Supervisor(fakeApi([state()]), workers, cfg);
    await sup.reconcile([target]);

    await sup.stopAll();

    expect(workers.terminated).toEqual(["a1"]);
  });
});
