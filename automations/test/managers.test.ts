import { describe, expect, spyOn, test } from "bun:test";
import type { AutomationsApi, ManagerReport } from "../src/automations-api.ts";
import { FakeResolver } from "../src/fake-resolver.ts";
import { WorkspaceManagers } from "../src/managers.ts";
import { OnePasswordResolver } from "../src/onepassword-resolver.ts";
import { TOKEN_ENV_PREFIX, takeWorkspaceTokens } from "../src/workspace-tokens.ts";
import { threadWorkerFactory } from "../src/worker.ts";

// The automation's own code, run inside the thread: it reports every variable it can see that holds a token.
const ENV_PROBE_BUNDLE = `import { parentPort } from "node:worker_threads";
const leaked = Object.entries(process.env).filter(([k, v]) => k.startsWith("NEXUL_OP_TOKEN_") || String(v).startsWith("ops_"));
parentPort.postMessage({ type: "log", level: "info", message: "env-probe:" + JSON.stringify(leaked) });
export default { name: "probe", description: "d", configSchema: {}, subscriptions: [], getHandler: () => undefined };`;

describe("takeWorkspaceTokens", () => {
  test("files each token under its workspace slug and leaves none in the environment", () => {
    const env: NodeJS.ProcessEnv = {
      [`${TOKEN_ENV_PREFIX}ACME`]: "ops_acme",
      [`${TOKEN_ENV_PREFIX}ACME_OPS_2`]: "ops_acme_ops",
      [`${TOKEN_ENV_PREFIX}`]: "ops_nameless",
      [`${TOKEN_ENV_PREFIX}_BAD`]: "ops_bad",
      NEXUL_SERVER_URL: "http://server",
    };

    const tokens = takeWorkspaceTokens(env);

    expect(Object.fromEntries(tokens)).toEqual({ acme: "ops_acme", "acme-ops-2": "ops_acme_ops" });
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

  test("opens one client per token and reports each workspace's state", async () => {
    const opened: string[] = [];
    const states = { "ops_acme": "ready", "ops_globex": "token rejected" } as const;
    const managers = new WorkspaceManagers(
      new Map([
        ["acme", "ops_acme"],
        ["globex", "ops_globex"],
      ]),
      (token) => (opened.push(token), new FakeResolver("op", {}, states[token as keyof typeof states])),
    );
    const sent: ManagerReport[][] = [];

    await managers.start();
    await managers.reportOnce(api(sent), "http://server", "nxa_1");

    expect(opened).toEqual(["ops_acme", "ops_globex"]);
    expect(sent).toEqual([
      [
        { workspace: "acme", manager: "op", state: "ready" },
        { workspace: "globex", manager: "op", state: "token rejected" },
      ],
    ]);
    expect(JSON.stringify(sent)).not.toContain("ops_");
  });

  test("with no token it still reports an empty list once, so a removed token clears, and a failed report is retried", async () => {
    const managers = new WorkspaceManagers(new Map(), () => new FakeResolver("op", {}));
    const sent: ManagerReport[][] = [];
    const server = api(sent, 1);

    await managers.start();
    await managers.reportOnce(server, "http://server", "nxa_1");
    expect(sent).toEqual([]);

    await managers.reportOnce(server, "http://server", "nxa_1");
    await managers.reportOnce(server, "http://server", "nxa_1");
    expect(sent).toEqual([[]]);
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
    ["a manager that cannot be reached", "dial tcp: lookup my.1password.com: no such host", "unreachable"],
    ["a failure that is not the network", "something else broke", "token rejected"],
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
