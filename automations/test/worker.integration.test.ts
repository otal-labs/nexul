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
});
