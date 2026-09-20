import { describe, expect, test } from "bun:test";
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
      },
      () => {
        stale = true;
      },
    );

    // Give the worker time to load the bundle, start DialinClient.run(),
    // and fail its first connect attempt — none of that should crash it.
    await new Promise((resolve) => setTimeout(resolve, 500));
    await handle.terminate();

    expect(stale).toBe(false);
  });
});
