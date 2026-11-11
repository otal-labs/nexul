import { describe, expect, spyOn, test } from "bun:test";
import { FakeResolver } from "./fake-resolver.ts";
import { WorkspaceManagers } from "../src/managers.ts";
import { PoolResolution } from "../src/pool-resolution.ts";
import { threadWorkerFactory } from "../src/worker.ts";

// Real worker_thread smoke test (the one piece the fake-based supervisor
// tests can't reach): a stub bundle plus an unreachable server URL should
// spawn and terminate cleanly without ever crashing the parent process,
// and terminate() must actually stop the thread before the test exits.
const STUB_BUNDLE = `export default {
  name: "smoke-test",
  description: "d",
  configSchema: {},
  subscriptions: [],
  getHandler: () => undefined,
};`;

describe("threadWorkerFactory (real worker_thread)", () => {
  test("spawns, runs briefly against an unreachable server, and terminates cleanly", async () => {
    let stale = false;
    const lines: string[] = [];
    let connectFailed: () => void = () => {};
    const failedFirstConnect = new Promise<void>((resolve) => {
      connectFailed = resolve;
    });
    const capture = (line: unknown) => {
      lines.push(String(line));
      if (String(line).includes("automation dial-in connection failed")) connectFailed();
    };
    const logSpy = spyOn(console, "log").mockImplementation(capture);
    const errorSpy = spyOn(console, "error").mockImplementation(capture);

    try {
      const handle = threadWorkerFactory.spawn(
        {
          automationId: "smoke-1",
          automationName: "smoke-test",
          code: STUB_BUNDLE,
          url: "http://127.0.0.1:1",
          token: "dat_smoke",
          timeoutMs: 30_000,
          heartbeatTimeoutMs: 60_000,
          memoryMb: 64,
          resolvePool: async () => ({}),
          onRunState: () => {},
        },
        () => {
          stale = true;
        },
      );

      await failedFirstConnect;
      await handle.terminate();
    } finally {
      logSpy.mockRestore();
      errorSpy.mockRestore();
    }

    expect(stale).toBe(false);
    expect(lines.filter((l) => l.includes("worker crashed") || l.includes("automation worker error") || l.includes("exited"))).toEqual([]);
  });

  test("a worker that exits on its own is reported stale, so the next poll respawns it", async () => {
    const lines: string[] = [];
    const logSpy = spyOn(console, "log").mockImplementation((line: unknown) => lines.push(String(line)));
    const errorSpy = spyOn(console, "error").mockImplementation((line: unknown) => lines.push(String(line)));

    try {
      await new Promise<void>((resolve) => {
        threadWorkerFactory.spawn(
          {
            automationId: "crash-1",
            automationName: "crash-test",
            code: "export const notDefault = 1;",
            url: "http://127.0.0.1:1",
            token: "dat_crash",
            timeoutMs: 30_000,
            heartbeatTimeoutMs: 60_000,
            memoryMb: 64,
            resolvePool: async () => ({}),
            onRunState: () => {},
          },
          resolve,
        );
      });
    } finally {
      logSpy.mockRestore();
      errorSpy.mockRestore();
    }

    expect(lines.some((l) => l.includes("automation worker exited on its own"))).toBe(true);
  });

  test("a thread's hello pool reaches its handler with references resolved by the supervisor and a failed one left out", async () => {
    const op = new FakeResolver("op", { "op://acme/db/password": "hunter2" });
    const managers = new WorkspaceManagers(new Map([["acme", "ops_acme"]]), () => op);
    await managers.start();
    const pool = new PoolResolution(managers, 0);
    const bundle = `export default {
      name: "pool-probe", description: "d", configSchema: {}, subscriptions: ["ticket.created"],
      getHandler: () => async (_payload, ctx) => {
        ctx.log("seen:" + JSON.stringify(Object.fromEntries(Object.entries(ctx.secrets).map(([k, v]) => [k, v.length]))));
        return true;
      },
    };`;
    let runLog: (log: string) => void = () => {};
    const seen = new Promise<string>((resolve) => {
      runLog = resolve;
    });
    const server = Bun.serve({
      port: 0,
      fetch: (req, srv) => (srv.upgrade(req) ? undefined : new Response("no", { status: 400 })),
      websocket: {
        message(ws, data) {
          const frame = JSON.parse(String(data)) as { type: string; log?: string };
          if (frame.type === "announce") {
            ws.send(JSON.stringify({ type: "hello", config_values: {}, secrets: { LIT: "plain", DB: "op://acme/db/password", GONE: "op://acme/db/missing" } }));
            ws.send(JSON.stringify({ type: "event", run_id: "r1", topic: "ticket.created", payload: {} }));
          }
          if (frame.type === "run_log") runLog(frame.log ?? "");
        },
      },
    });
    const logSpy = spyOn(console, "log").mockImplementation(() => {});
    const errorSpy = spyOn(console, "error").mockImplementation(() => {});

    try {
      const handle = threadWorkerFactory.spawn(
        {
          automationId: "pool-1",
          automationName: "pool-probe",
          code: bundle,
          url: `http://127.0.0.1:${server.port}`,
          token: "dat_pool",
          timeoutMs: 30_000,
          heartbeatTimeoutMs: 60_000,
          memoryMb: 64,
          resolvePool: (entries) => pool.resolve({ workspaceId: "acme", automationId: "pool-1", automationName: "pool-probe" }, entries),
          onRunState: () => {},
        },
        () => {},
      );
      const line = await seen;
      await handle.terminate();
      expect((JSON.parse(line) as { message: string }).message).toBe('seen:{"LIT":5,"DB":7}');
      expect(line).not.toContain("hunter2");
    } finally {
      logSpy.mockRestore();
      errorSpy.mockRestore();
      await server.stop(true);
    }
  });
});
