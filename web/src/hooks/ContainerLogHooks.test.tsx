import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useContainerLogs } from "@/hooks/ContainerLogHooks";
import { MAX_LOG_LINES } from "@/models/ContainerLog";
import { useSessionStore } from "@/stores/sessionStore";
import { useViewAsStore } from "@/stores/viewAsStore";

describe("useContainerLogs", () => {
  const sockets = vi.fn();

  beforeEach(() => {
    sockets.mockClear();
    vi.stubGlobal("WebSocket", class { constructor(url: string) { sockets(url); } close() {} });
    useSessionStore.setState({ token: "tok-1", isLoggedIn: true });
    useViewAsStore.getState().exit();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("follows the tail with the member's own session", () => {
    const { result } = renderHook(() => useContainerLogs("/stacks/s-1/services/web/logs"));
    expect(sockets).toHaveBeenCalledTimes(1);
    expect(result.current.status).toBe("connecting");
  });

  it("opens no socket while viewing as another member, since it would read with the signed-in member's access", () => {
    useViewAsStore.getState().start({ workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "" });
    const { result } = renderHook(() => useContainerLogs("/stacks/s-1/services/web/logs"));
    expect(sockets).not.toHaveBeenCalled();
    expect(result.current).toMatchObject({ status: "forbidden", reason: "Live logs are off while viewing as another member." });
  });

  it("keeps every line of a burst in order, numbered without gaps, and only the newest 5000", async () => {
    const frames: FrameRequestCallback[] = [];
    const socket: { onmessage?: (ev: { data: string }) => void } = {};
    vi.stubGlobal("WebSocket", class {
      set onmessage(handler: (ev: { data: string }) => void) {
        socket.onmessage = handler;
      }
      close() {}
    });
    vi.stubGlobal("requestAnimationFrame", (run: FrameRequestCallback) => frames.push(run));
    vi.stubGlobal("cancelAnimationFrame", () => undefined);
    const { result } = renderHook(() => useContainerLogs("/stacks/s-1/services/web/logs"));

    const total = MAX_LOG_LINES + 500;
    for (let first = 0; first < total; first += 5) {
      const lines = Array.from({ length: 5 }, (_, i) => ({ ts: `t${first + i}`, stream: "stdout", line: `n${first + i}` }));
      socket.onmessage?.({ data: JSON.stringify({ lines }) });
    }
    expect(frames).toHaveLength(1);
    await act(async () => frames.splice(0).forEach((run) => run(0)));

    const kept = result.current.lines;
    expect(kept).toHaveLength(MAX_LOG_LINES);
    expect(kept.map((l) => l.text)).toEqual(Array.from({ length: MAX_LOG_LINES }, (_, i) => `n${i + 500}`));
    expect(kept.every((l, i) => i === 0 || l.seq === kept[i - 1]!.seq + 1)).toBe(true);
  });
});
