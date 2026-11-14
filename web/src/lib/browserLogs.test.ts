import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { installBrowserLogs } from "@/lib/browserLogs";
import { useSessionStore } from "@/stores/sessionStore";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() } }));

const post = vi.mocked(api.post);

// Stands in for the browser's observer; jsdom has none, and no test browser can stall a frame on demand.
class fakeObserver {
  static supportedEntryTypes = ["long-animation-frame"];
  static emit: (entries: unknown[]) => void = () => undefined;
  constructor(callback: (list: { getEntries: () => unknown[] }) => void) {
    fakeObserver.emit = (entries) => callback({ getEntries: () => entries });
  }
  observe() {}
}

describe("installBrowserLogs", () => {
  const originalError = console.error;
  const originalWarn = console.warn;
  let errorSpy: ReturnType<typeof vi.fn<(...data: unknown[]) => void>>;

  beforeEach(() => {
    vi.useFakeTimers();
    post.mockResolvedValue({});
    useSessionStore.getState().login("token");
    errorSpy = vi.fn();
    console.error = errorSpy;
    console.warn = vi.fn();
    installBrowserLogs();
  });

  afterEach(() => {
    console.error = originalError;
    console.warn = originalWarn;
    useSessionStore.getState().logout();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("batches console errors, warnings, and uncaught errors into one relay call", () => {
    console.error("boom", new Error("bad"));
    console.warn("ws dropped malformed frame", { url: "ws://x" });
    window.dispatchEvent(new ErrorEvent("error", { message: "uncaught", filename: "app.js", lineno: 7 }));

    expect(post).not.toHaveBeenCalled();
    vi.advanceTimersByTime(2_000);

    expect(post).toHaveBeenCalledTimes(1);
    const { records } = post.mock.calls[0]?.[1] as { records: { level: string; message: string; attrs?: unknown }[] };
    expect(records.map((r) => [r.level, r.message])).toEqual([
      ["error", "boom Error: bad"],
      ["warn", 'ws dropped malformed frame {"url":"ws://x"}'],
      ["error", "uncaught"],
    ]);
    expect(records[0]?.attrs).toMatchObject({ stack: expect.stringContaining("bad") });
    expect(records[2]?.attrs).toMatchObject({ file: "app.js", line: 7 });
  });

  it("reports a frame that stalls the main thread for seconds, naming its slowest scripts, and ignores shorter frames", () => {
    const script = (duration: number, fn: string) => ({
      invoker: "DIV.onclick", sourceURL: "https://example.com/app.js", sourceFunctionName: fn,
      sourceCharPosition: 42, duration, forcedStyleAndLayoutDuration: 5,
    });
    const frame = (duration: number) => ({
      startTime: 100, duration, blockingDuration: duration - 50, styleAndLayoutStart: 100 + duration - 300,
      scripts: [script(10, "tiny"), script(900, "slow"), script(20, "small"), script(400, "slower")],
    });
    vi.stubGlobal("PerformanceObserver", fakeObserver);
    installBrowserLogs();
    fakeObserver.emit([frame(1_999), frame(4_000)]);
    vi.advanceTimersByTime(2_000);

    const { records } = post.mock.calls[0]?.[1] as { records: { level: string; message: string; attrs?: unknown }[] };
    expect(records).toEqual([{
      level: "warn",
      message: "main thread stalled for 4000ms",
      url: "/",
      attrs: {
        duration_ms: 4_000, blocking_ms: 3_950, layout_ms: 300, script_ms: 1_330,
        script_1: "900ms DIV.onclick → slow https://example.com/app.js:42 (forced layout 5ms)",
        script_2: "400ms DIV.onclick → slower https://example.com/app.js:42 (forced layout 5ms)",
        script_3: "20ms DIV.onclick → small https://example.com/app.js:42 (forced layout 5ms)",
      },
    }]);
    vi.unstubAllGlobals();
  });

  it("holds records until a session exists and never surfaces relay failures", async () => {
    useSessionStore.getState().logout();
    console.error("before login");
    vi.advanceTimersByTime(2_000);
    expect(post).not.toHaveBeenCalled();

    post.mockRejectedValueOnce(new Error("offline"));
    useSessionStore.getState().login("token");
    console.error("after login");
    vi.advanceTimersByTime(2_000);
    expect(post).toHaveBeenCalledTimes(1);
    await vi.runAllTimersAsync();
    // Both calls are the originals we forwarded; a relay failure must never add a third.
    expect(errorSpy).toHaveBeenCalledTimes(2);
  });
});
