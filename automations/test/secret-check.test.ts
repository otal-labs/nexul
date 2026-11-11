import { describe, expect, spyOn, test } from "bun:test";
import { existsSync, mkdtempSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { AutomationTarget, AutomationsApi, RemoteState } from "../src/automations-api.ts";
import { loadConfig } from "../src/config.ts";
import { FakeResolver } from "../src/fake-resolver.ts";
import { WorkspaceManagers } from "../src/managers.ts";
import { PoolResolution } from "../src/pool-resolution.ts";
import { parseCheckInterval, SecretCheck, type Clock } from "../src/secret-check.ts";
import { Fingerprinter, loadFingerprinter } from "../src/secret-fingerprints.ts";
import { Supervisor } from "../src/supervisor.ts";
import type { SpawnOptions, WorkerFactory, WorkerHandle } from "../src/worker.ts";

const DB = "op://acme/db/password";
const API = "op://acme/api/key";
const GLOBEX_KEY = "op://globex/api/key";
const MINUTE = 60_000;

const acme = (id: string): AutomationTarget => ({ id, name: `Automation ${id}`, token: `dat_${id}`, workspace_id: "acme", workspace: "acme" });
const globex: AutomationTarget = { id: "g1", name: "Deploy notice", token: "dat_g1", workspace_id: "globex", workspace: "globex" };
const cfg = { serverUrl: "http://server", timeoutMs: 30_000, heartbeatTimeoutMs: 90_000, memoryMb: 128 };
const state: RemoteState = { enabled: true, configValues: "{}", activeVersionId: "v1", activeCode: "export default {}" };
const api: AutomationsApi = { fetchAssignments: async () => ({ removed: false, automations: [] }), reportManagers: async () => {}, fetchState: async () => state };

function fakeClock(): Clock & { advance(ms: number): void; pending(): number } {
  let now = 0;
  let timers: { at: number; fn: () => void }[] = [];
  return {
    after(ms, fn) {
      const timer = { at: now + ms, fn };
      timers.push(timer);
      return () => {
        timers = timers.filter((t) => t !== timer);
      };
    },
    advance(ms) {
      const to = now + ms;
      for (;;) {
        const due = timers.filter((t) => t.at <= to).sort((a, b) => a.at - b.at)[0];
        if (!due) break;
        timers = timers.filter((t) => t !== due);
        now = due.at;
        due.fn();
      }
      now = to;
    },
    pending: () => timers.length,
  };
}

function fakeWorkers(): WorkerFactory & { spawns: SpawnOptions[]; terminated: string[] } {
  const spawns: SpawnOptions[] = [];
  const terminated: string[] = [];
  return {
    spawns,
    terminated,
    spawn(opts): WorkerHandle {
      spawns.push(opts);
      return {
        async terminate() {
          terminated.push(opts.automationId);
        },
      };
    },
  };
}

const settle = () => new Promise<void>((resolve) => setImmediate(resolve));

// A host with one fake manager per workspace; `values` is what each manager currently answers, edited by the tests.
async function rig(targets: AutomationTarget[], values: Record<string, Record<string, string>>) {
  const resolvers = Object.fromEntries(Object.entries(values).map(([workspace, entries]) => [workspace, new FakeResolver("op", entries)]));
  const managers = new WorkspaceManagers(new Map(Object.keys(values).map((w) => [w, `ops_${w}`])), (token) => resolvers[token.slice(4)] as FakeResolver);
  managers.start();
  const clock = fakeClock();
  const fingerprinter = new Fingerprinter(Buffer.alloc(32, 7));
  const workers = fakeWorkers();
  const supervisor = new Supervisor(api, workers, cfg, new PoolResolution(managers, 0), { fingerprinter, clock });
  await supervisor.reconcile(targets);
  // Each worker connects: the pool it asks for is its automation's references, resolved at its hello.
  const connect = async (index: number, pool: Record<string, string>) => (workers.spawns[index] as SpawnOptions).resolvePool(pool);
  return { resolvers, managers, clock, fingerprinter, workers, supervisor, connect };
}

describe("secret check restarts", () => {
  test("an idle automation restarts at once when a value changed, and reconnects with the new one", async () => {
    const values = { acme: { [DB]: "hunter2-old" } };
    const h = await rig([acme("a1")], values);
    await h.connect(0, { DB });

    values.acme[DB] = "hunter2-new";
    await h.supervisor.checkSecrets(h.managers);
    await settle();

    expect(h.workers.terminated).toEqual(["a1"]);
    expect(h.workers.spawns).toHaveLength(2);
    expect(await h.connect(1, { DB })).toEqual({ DB: "hunter2-new" });
  });

  test("an automation whose values did not change is left running", async () => {
    const h = await rig([acme("a1")], { acme: { [DB]: "hunter2-old" } });
    await h.connect(0, { DB });

    await h.supervisor.checkSecrets(h.managers);
    await settle();

    expect(h.workers.terminated).toEqual([]);
    expect(h.workers.spawns).toHaveLength(1);
  });

  test("a value that cannot be read now does not restart anything", async () => {
    const values: Record<string, Record<string, string>> = { acme: { [DB]: "hunter2-old" } };
    const h = await rig([acme("a1")], values);
    await h.connect(0, { DB });

    delete values.acme?.[DB];
    await h.supervisor.checkSecrets(h.managers);
    await settle();

    expect(h.workers.terminated).toEqual([]);
  });

  test("a busy automation restarts when its run ends, not before", async () => {
    const values = { acme: { [DB]: "hunter2-old" } };
    const h = await rig([acme("a1")], values);
    await h.connect(0, { DB });
    (h.workers.spawns[0] as SpawnOptions).onRunState(true);

    values.acme[DB] = "hunter2-new";
    await h.supervisor.checkSecrets(h.managers);
    await settle();
    expect(h.workers.terminated).toEqual([]);

    (h.workers.spawns[0] as SpawnOptions).onRunState(false);
    await settle();

    expect(h.workers.terminated).toEqual(["a1"]);
    expect(h.workers.spawns).toHaveLength(2);
    h.clock.advance(10 * MINUTE);
    expect(h.workers.spawns).toHaveLength(2);
  });

  test("a run still in flight after 5 minutes is cut off and the automation restarts", async () => {
    const values = { acme: { [DB]: "hunter2-old" } };
    const h = await rig([acme("a1")], values);
    await h.connect(0, { DB });
    (h.workers.spawns[0] as SpawnOptions).onRunState(true);

    values.acme[DB] = "hunter2-new";
    await h.supervisor.checkSecrets(h.managers);
    h.clock.advance(5 * MINUTE - 1);
    await settle();
    expect(h.workers.terminated).toEqual([]);

    h.clock.advance(1);
    await settle();

    expect(h.workers.terminated).toEqual(["a1"]);
    expect(h.workers.spawns).toHaveLength(2);
  });

  test("an automation switched off while waiting for its run to end is not brought back", async () => {
    const values = { acme: { [DB]: "hunter2-old" } };
    const h = await rig([acme("a1")], values);
    await h.connect(0, { DB });
    (h.workers.spawns[0] as SpawnOptions).onRunState(true);
    values.acme[DB] = "hunter2-new";
    await h.supervisor.checkSecrets(h.managers);

    await h.supervisor.reconcile([]);
    h.clock.advance(10 * MINUTE);
    await settle();

    expect(h.workers.spawns).toHaveLength(1);
  });
});

describe("secret check reads", () => {
  test("one read per workspace each interval, however many automations hold references", async () => {
    const h = await rig([acme("a1"), acme("a2"), globex], { acme: { [DB]: "d", [API]: "k" }, globex: { [GLOBEX_KEY]: "g" } });
    await h.connect(0, { DB, API });
    await h.connect(1, { DB });
    await h.connect(2, { KEY: GLOBEX_KEY });
    const acmeReads = (h.resolvers.acme as FakeResolver).calls;
    const globexReads = (h.resolvers.globex as FakeResolver).calls;
    acmeReads.length = 0;
    globexReads.length = 0;

    const check = new SecretCheck(h.supervisor, h.managers, 30 * MINUTE, h.clock);
    check.start();
    h.clock.advance(30 * MINUTE);
    await settle();
    expect([acmeReads.length, globexReads.length]).toEqual([1, 1]);
    expect(acmeReads[0]?.slice().sort()).toEqual([API, DB]);

    h.clock.advance(30 * MINUTE);
    await settle();
    expect([acmeReads.length, globexReads.length]).toEqual([2, 2]);
  });

  test("off schedules nothing and reads nothing", async () => {
    const h = await rig([acme("a1")], { acme: { [DB]: "d" } });
    await h.connect(0, { DB });
    const reads = (h.resolvers.acme as FakeResolver).calls;
    reads.length = 0;

    new SecretCheck(h.supervisor, h.managers, parseCheckInterval("off"), h.clock).start();
    h.clock.advance(24 * 60 * MINUTE);
    await settle();

    expect(h.clock.pending()).toBe(0);
    expect(reads).toEqual([]);
  });
});

describe("parseCheckInterval", () => {
  test.each([
    [undefined, 60 * MINUTE],
    ["", 60 * MINUTE],
    ["off", 0],
    ["OFF", 0],
    ["5m", 5 * MINUTE],
    ["30m", 30 * MINUTE],
    ["1h30m", 90 * MINUTE],
  ])("%p reads as %p ms", (raw, ms) => {
    expect(parseCheckInterval(raw)).toBe(ms);
  });

  test.each(["4m59s", "1m", "30s", "0", "soon", "10 minutes", "-5m"])("%p is refused", (raw) => {
    expect(() => parseCheckInterval(raw)).toThrow("NEXUL_SECRET_CHECK_INTERVAL");
  });

  test("an interval under 5m stops the host at start", () => {
    const env = { NEXUL_SERVER_URL: "http://s", NEXUL_CREDENTIAL_FILE: "/c", NEXUL_AUTOMATIONS_HOST_NAME: "h", NEXUL_SECRET_CHECK_INTERVAL: "1m" };
    expect(() => loadConfig(env)).toThrow("under the 5m minimum");
    expect(loadConfig({ ...env, NEXUL_SECRET_CHECK_INTERVAL: "off" }).secretCheckIntervalMs).toBe(0);
  });
});

describe("secret check logging", () => {
  test("nothing logged holds a value, a reference or a fingerprint", async () => {
    const values = { acme: { [DB]: "hunter2-old", [API]: "key-old" } };
    const h = await rig([acme("a1")], values);
    const lines: string[] = [];
    const capture = (line: unknown) => void lines.push(String(line));
    const logSpy = spyOn(console, "log").mockImplementation(capture);
    const errorSpy = spyOn(console, "error").mockImplementation(capture);

    try {
      await h.connect(0, { DB, API });
      values.acme[DB] = "hunter2-new";
      await h.supervisor.checkSecrets(h.managers);
      await settle();
    } finally {
      logSpy.mockRestore();
      errorSpy.mockRestore();
    }

    expect(lines.some((l) => l.includes('"automationId":"a1"') && l.includes('"changed":1'))).toBe(true);
    const secrets = ["hunter2-old", "hunter2-new", "key-old", DB, API, "op://", h.fingerprinter.of("hunter2-old"), h.fingerprinter.of("hunter2-new"), h.fingerprinter.of("key-old")];
    for (const secret of secrets) expect(lines.filter((l) => l.includes(secret))).toEqual([]);
  });
});

describe("loadFingerprinter", () => {
  test("makes a 0600 key once and gives the same fingerprints from it afterwards", () => {
    const file = join(mkdtempSync(join(tmpdir(), "fingerprint-key-")), "state", "secret-check.key");

    const first = loadFingerprinter(file);
    const second = loadFingerprinter(file);

    expect(first.persisted).toBe(true);
    expect(statSync(file).mode & 0o777).toBe(0o600);
    expect(second.fingerprinter.of("x")).toBe(first.fingerprinter.of("x"));
  });

  test("a directory that cannot hold the key leaves an in-memory one and says so", () => {
    const dir = mkdtempSync(join(tmpdir(), "fingerprint-key-"));
    const file = join(dir, "secret-check.key", "inside");
    writeFileSync(join(dir, "secret-check.key"), "a file where a directory is needed");

    const loaded = loadFingerprinter(file);

    expect(loaded.persisted).toBe(false);
    expect(existsSync(file)).toBe(false);
    expect(loaded.fingerprinter.of("x")).toHaveLength(64);
  });
});
