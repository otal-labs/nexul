import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { installBrowserLogs } from "@/lib/browserLogs";
import { useSessionStore } from "@/stores/sessionStore";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() } }));

const post = vi.mocked(api.post);

describe("installBrowserLogs", () => {
  const originalError = console.error;
  const originalWarn = console.warn;
  let errorSpy: ReturnType<typeof vi.fn>;

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
