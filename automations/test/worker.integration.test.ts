import { describe, expect, spyOn, test } from "bun:test";
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
    expect(lines.filter((l) => l.includes("worker crashed") || l.includes("automation worker error"))).toEqual([]);
  });
});
