import { describe, expect, spyOn, test } from "bun:test";
import type { AutomationsApi, ManagerReport } from "../src/automations-api.ts";
import { FakeResolver } from "../src/fake-resolver.ts";
import { pollOnce } from "../src/host.ts";
import { WorkspaceManagers, type ManagerTimers } from "../src/managers.ts";
import { OnePasswordResolver } from "../src/onepassword-resolver.ts";
import type { ManagerState, ResolveResult, Resolver } from "../src/resolver.ts";
import { TOKEN_ENV_PREFIX, takeWorkspaceTokens } from "../src/workspace-tokens.ts";
import { threadWorkerFactory } from "../src/worker.ts";

// The automation's own code, run inside the thread: it reports every variable it can see that holds a token.
const ENV_PROBE_BUNDLE = `import { parentPort } from "node:worker_threads";
const leaked = Object.entries(process.env).filter(([k, v]) => k.startsWith("NEXUL_OP_TOKEN_") || String(v).startsWith("ops_"));
parentPort.postMessage({ type: "log", level: "info", message: "env-probe:" + JSON.stringify(leaked) });
export default { name: "probe", description: "d", configSchema: {}, subscriptions: [], getHandler: () => undefined };`;

describe("takeWorkspaceTokens", () => {
  // The same names internal/install's TestWorkspaceTokenEnv_RoundTripsAnyID writes.
  test("files each token under the workspace id its name encodes and leaves none in the environment", () => {
    const env: NodeJS.ProcessEnv = {
      [`${TOKEN_ENV_PREFIX}776F726B73706163652D64656661756C74`]: "ops_default",
      [`${TOKEN_ENV_PREFIX}30313961366633652D376331642D376232612D396630302D316332643365346635613662`]: "ops_globex",
      [`${TOKEN_ENV_PREFIX}`]: "ops_nameless",
      [`${TOKEN_ENV_PREFIX}ACME`]: "ops_slug",
      [`${TOKEN_ENV_PREFIX}abcd`]: "ops_lower",
      NEXUL_SERVER_URL: "http://server",
    };

    const tokens = takeWorkspaceTokens(env);

    expect(Object.fromEntries(tokens)).toEqual({
      "workspace-default": "ops_default",
      "019a6f3e-7c1d-7b2a-9f00-1c2d3e4f5a6b": "ops_globex",
    });
    expect(env).toEqual({ NEXUL_SERVER_URL: "http://server" });
  });

  test("an automation thread started after the tokens were taken sees none of them", async () => {
    process.env[`${TOKEN_ENV_PREFIX}ACME`] = "ops_acme";
    takeWorkspaceTokens();
    const lines: string[] = [];
    let seen: () => void = () => {};
    const probed = new Promise<void>((resolve) => {
      seen = resolve;
    });
    const capture = (line: unknown) => {
      lines.push(String(line));
      if (String(line).includes("env-probe:")) seen();
    };
    const logSpy = spyOn(console, "log").mockImplementation(capture);
    const errorSpy = spyOn(console, "error").mockImplementation(capture);

    try {
      const handle = threadWorkerFactory.spawn(
        {
          automationId: "probe-1",
          automationName: "probe",
          code: ENV_PROBE_BUNDLE,
          url: "http://127.0.0.1:1",
          token: "dat_probe",
          timeoutMs: 30_000,
          heartbeatTimeoutMs: 60_000,
          memoryMb: 64,
          resolvePool: async () => ({}),
          onRunState: () => {},
        },
        () => {},
      );
      await probed;
      await handle.terminate();
    } finally {
      logSpy.mockRestore();
      errorSpy.mockRestore();
    }

    const probe = lines.find((l) => l.includes("env-probe:")) ?? "";
    expect(probe).toContain("env-probe:[]");
    expect(lines.join("\n")).not.toContain("ops_acme");
  });
});

describe("WorkspaceManagers", () => {
  function api(sent: ManagerReport[][], failures = 0): AutomationsApi {
    let remaining = failures;
    return {
      fetchAssignments: async () => ({ removed: false, automations: [] }),
      fetchState: async () => {
        throw new Error("not used");
      },
      reportManagers: async (_url, _credential, managers) => {
        if (remaining-- > 0) throw new Error("server down");
        sent.push(managers);
      },
    };
  }

  // Time stands still until a test releases a delay by its length; no jitter, so a retry wait is exactly its base.
  function fakeTimers() {
    const pending: { ms: number; release: () => void }[] = [];
    const timers: ManagerTimers = {
      delay: (ms) => new Promise<void>((release) => pending.push({ ms, release })),
      random: () => 0,
    };
    const release = async (ms: number): Promise<void> => {
      const i = pending.findIndex((p) => p.ms === ms);
      expect(i).toBeGreaterThanOrEqual(0);
      pending.splice(i, 1)[0]?.release();
      await drain();
    };
    return { timers, pending, release };
  }
  const drain = () => new Promise<void>((resolve) => setImmediate(resolve));

  class Scripted implements Resolver {
    readonly scheme = "op";
    calls = 0;
    constructor(private readonly script: (call: number) => Promise<ManagerState>) {}
    check(): Promise<ManagerState> {
      return this.script(++this.calls);
    }
    async resolveAll(): Promise<Map<string, ResolveResult>> {
      return new Map();
    }
  }
  const tokens = (...workspaces: string[]) => new Map(workspaces.map((w) => [w, `ops_${w}`]));

  test("opens one client per token and reports each workspace's state", async () => {
    const opened: string[] = [];
    const states = { "ops_acme": "ready", "ops_globex": "token rejected" } as const;
    const managers = new WorkspaceManagers(
      tokens("acme", "globex"),
      (token) => (opened.push(token), new FakeResolver("op", {}, states[token as keyof typeof states])),
      fakeTimers().timers,
    );
    const sent: ManagerReport[][] = [];

    managers.start();
    await drain();
    await managers.report(api(sent), "http://server", "nxa_1");

    expect(opened).toEqual(["ops_acme", "ops_globex"]);
    expect(sent).toEqual([
      [
        { workspace_id: "acme", manager: "op", state: "ready" },
        { workspace_id: "globex", manager: "op", state: "token rejected" },
      ],
    ]);
    expect(JSON.stringify(sent)).not.toContain("ops_");
  });

  test.each([
    ["no token runs every workspace's automations", [] as string[], ["a-acme", "a-globex"]],
    ["one workspace's token runs only that workspace's automations", ["ws-acme"], ["a-acme"]],
    ["tokens of two workspaces run none", ["ws-acme", "ws-globex"], []],
  ])("%s", (_name, held, want) => {
    const managers = new WorkspaceManagers(new Map(held.map((id) => [id, `ops_${id}`])), () => new FakeResolver("op", {}));
    const targets = [
      { id: "a-acme", name: "a", token: "dat_a", workspace_id: "ws-acme", workspace: "acme" },
      { id: "a-globex", name: "b", token: "dat_b", workspace_id: "ws-globex", workspace: "globex" },
    ];

    expect(managers.servable(targets).map((t) => t.id)).toEqual(want);
  });

  test("with no token it still reports an empty list once, so a removed token clears, and a failed report is retried", async () => {
    const managers = new WorkspaceManagers(new Map(), () => new FakeResolver("op", {}), fakeTimers().timers);
    const sent: ManagerReport[][] = [];
    const server = api(sent, 1);

    managers.start();
    await managers.report(server, "http://server", "nxa_1");
    expect(sent).toEqual([]);

    await managers.report(server, "http://server", "nxa_1");
    await managers.report(server, "http://server", "nxa_1");
    expect(sent).toEqual([[]]);
  });

  test("an unreachable manager is retried with doubling backoff up to the cap, and the recovery is reported again", async () => {
    const { timers, release } = fakeTimers();
    const resolver = new Scripted(async (call) => (call <= 8 ? "unreachable" : "ready"));
    const managers = new WorkspaceManagers(tokens("acme"), () => resolver, timers);
    const sent: ManagerReport[][] = [];
    const server = api(sent);

    managers.start();
    await drain();
    await managers.report(server, "http://server", "nxa_1");
    await managers.report(server, "http://server", "nxa_1");
    expect(sent).toEqual([[{ workspace_id: "acme", manager: "op", state: "unreachable" }]]);

    for (const wait of [5_000, 10_000, 20_000, 40_000, 80_000, 160_000, 300_000, 300_000]) await release(wait);
    await managers.report(server, "http://server", "nxa_1");

    expect(resolver.calls).toBe(9);
    expect(sent.at(-1)).toEqual([{ workspace_id: "acme", manager: "op", state: "ready" }]);
    expect(sent).toHaveLength(2);
  });

  test("a rejected token is reported once and never tried again", async () => {
    const { timers, pending } = fakeTimers();
    const resolver = new Scripted(async () => "token rejected");
    const managers = new WorkspaceManagers(tokens("acme"), () => resolver, timers);
    const sent: ManagerReport[][] = [];

    managers.start();
    await drain();
    await managers.report(api(sent), "http://server", "nxa_1");

    expect(sent).toEqual([[{ workspace_id: "acme", manager: "op", state: "token rejected" }]]);
    expect(pending.filter((p) => p.ms !== 30_000)).toEqual([]);
    expect(resolver.calls).toBe(1);
  });

  test("polling does not wait on a check that hangs; once its time limit passes the manager reads unreachable", async () => {
    const { timers, release } = fakeTimers();
    const hung = new Scripted(() => new Promise<ManagerState>(() => {}));
    const managers = new WorkspaceManagers(tokens("acme"), () => hung, timers);
    const sent: ManagerReport[][] = [];
    const server = api(sent);
    const poll = { api: server, supervisor: { reconcile: async () => {} }, managers, serverUrl: "http://server", credential: "nxa_1" };

    managers.start();
    const polled = await pollOnce(poll);
    expect(polled).toBe("running");
    expect(sent).toEqual([]);

    await release(30_000);
    await pollOnce(poll);

    expect(sent).toEqual([[{ workspace_id: "acme", manager: "op", state: "unreachable" }]]);
  });
});

// The same messages and states are tested for the runner and server in internal/platform/secretref/onepassword.
describe("WorkspaceManagers.resolve", () => {
  const drain = () => new Promise<void>((resolve) => setImmediate(resolve));
  const timers: ManagerTimers = { delay: () => new Promise<void>(() => {}), random: () => 0 };
  const server = (sent: ManagerReport[][]): AutomationsApi => ({
    fetchAssignments: async () => ({ removed: false, automations: [] }),
    fetchState: async () => {
      throw new Error("not used");
    },
    reportManagers: async (_url, _credential, managers) => {
      sent.push(managers.map((m) => ({ ...m })));
    },
  });

  test("the reported state follows what resolving shows: unreachable until a read gets through, then ready, and back", async () => {
    const entries: Record<string, string | { error: "unreachable" }> = { "op://a/b/c": { error: "unreachable" } };
    const managers = new WorkspaceManagers(new Map([["ws-1", "ops_acme"]]), () => new FakeResolver("op", entries, "ready"), timers);
    const sent: ManagerReport[][] = [];
    managers.start();
    await drain();

    await managers.resolve("ws-1", ["op://a/b/c"]);
    await managers.report(server(sent), "http://server", "nxa_1");
    entries["op://a/b/c"] = "value";
    await managers.resolve("ws-1", ["op://a/b/c"]);
    await managers.report(server(sent), "http://server", "nxa_1");
    await managers.report(server(sent), "http://server", "nxa_1");

    expect(sent.map((report) => report[0]?.state)).toEqual(["unreachable", "ready"]);
  });

  test("a rejected token stays rejected whatever a resolve returns", async () => {
    const managers = new WorkspaceManagers(
      new Map([["ws-1", "ops_acme"]]),
      () => new FakeResolver("op", { "op://a/b/c": { error: "unreachable" } }, "token rejected"),
      timers,
    );
    const sent: ManagerReport[][] = [];
    managers.start();
    await drain();

    await managers.resolve("ws-1", ["op://a/b/c"]);
    await managers.report(server(sent), "http://server", "nxa_1");

    expect(sent.map((report) => report[0]?.state)).toEqual(["token rejected"]);
  });
});

describe("OnePasswordResolver.check", () => {
  const failing = (message: string) =>
    new OnePasswordResolver("ops_secret", async () => {
      throw new Error(message);
    });

  test.each([
    ["a malformed token", "invalid user input: encountered the following errors: service account token had invalid format", "token rejected"],
    ["a token the account refuses", "invalid service account token, please make sure you provide a valid service account token", "token rejected"],
    ["a 401", "request failed: 401 Unauthorized", "token rejected"],
    ["a 403", "request failed: 403 Forbidden", "token rejected"],
    ["a token that cannot be decoded", "failed to deserialize service account token", "token rejected"],
    ["a manager that cannot be reached", "dial tcp: lookup my.1password.com: no such host", "unreachable"],
    ["a rate limit", "429 too many requests", "unreachable"],
    ["a server error", "503 service unavailable", "unreachable"],
    ["a TLS failure", "x509: certificate signed by unknown authority", "unreachable"],
    ["a failure nobody recognises", "something else broke", "unreachable"],
  ] as const)("%s is %s", async (_name, message, state) => {
    expect(await failing(message).check()).toBe(state);
  });

  test("a client that opens is ready, and is the one a later resolve reuses", async () => {
    let opened = 0;
    const resolver = new OnePasswordResolver("ops_secret", async () => {
      opened++;
      return { secrets: { resolveAll: async () => ({ individualResponses: {} }) } } as never;
    });

    expect(await resolver.check()).toBe("ready");
    await resolver.resolveAll(["op://a/b/c"]);

    expect(opened).toBe(1);
  });
});
