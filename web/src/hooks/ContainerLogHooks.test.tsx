import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useContainerLogs } from "@/hooks/ContainerLogHooks";
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
});
