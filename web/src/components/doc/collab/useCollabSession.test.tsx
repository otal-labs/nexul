import { StrictMode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Awareness, encodeAwarenessUpdate } from "y-protocols/awareness";
import { Doc } from "yjs";

import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import type { LiveSocket } from "@/api/ws";

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  dispatch(raw: string) {
    this.onmessage?.({ data: raw });
  }
}

const framesOf = (socket: FakeSocket, type: string) =>
  socket.send.mock.calls
    .map((c) => JSON.parse(c[0] as string) as { type: string })
    .filter((f) => f.type === type);

const makeSocket = () => {
  const socket = new FakeSocket();
  return { socket, factory: () => socket };
};

// connect() is deferred by a tick (ws-25, see useCollabSession.ts), so tests must let that tick pass before a wsFactory-produced socket has its handlers wired up.
const flushConnect = () => act(() => new Promise((r) => setTimeout(r, 0)));

const remotePresence = () => {
  const remote = new Awareness(new Doc());
  remote.clientID = 7;
  remote.setLocalState({ user: { name: "Bob", color: "#22c55e", activity: "viewing" } });
  remote.setLocalState({ user: { name: "Bob", color: "#22c55e", activity: "viewing" } });
  return btoa(String.fromCharCode(...encodeAwarenessUpdate(remote, [7])));
};

describe("useCollabSession", () => {
  it("stays null and never opens a socket without a token (buildCollabURL requires one)", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", null, { wsFactory: factory }));
    expect(result.current).toBeNull();
    await flushConnect();
    expect(socket.send).not.toHaveBeenCalled();
  });

  it("creates the session with a colored presence for the acting user", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }));
    expect(result.current).not.toBeNull();
    const session = result.current!;
    expect(session.user.name).toBe("Alice");
    expect(session.user.activity).toBe("editing");
    expect(session.user.color).toMatch(/^#/);

    await flushConnect();
    act(() => socket.onopen?.({}));
    await waitFor(() => expect(framesOf(socket, "hello")).toHaveLength(1));
  });

  it("derives participants from relayed awareness (viewers included)", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "view", "Alice", "token-1", { wsFactory: factory }));
    await flushConnect();
    act(() => socket.onopen?.({}));
    socket.dispatch(JSON.stringify({ type: "presence", from: "bob", client_id: 7, payload: remotePresence() }));

    await waitFor(() => {
      expect(result.current?.participants.some((p) => p.name === "Bob" && p.activity === "viewing")).toBe(true);
    });
  });

  it("flags unresolvable payloads and resets to a fresh session", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }));
    const first = result.current!;
    await flushConnect();
    act(() => socket.onopen?.({}));

    act(() => socket.dispatch(JSON.stringify({ type: "update", from: "bob", seq: 1, payload: btoa("		") })));
    await waitFor(() => expect(result.current?.applyError).toBe(true));

    act(() => result.current?.dismissApplyError());
    expect(result.current?.applyError).toBe(false);

    act(() => result.current?.resetSession());
    await waitFor(() => expect(result.current).not.toBeNull());
    expect(result.current?.doc).not.toBe(first.doc);
  });

  it("feeds the registered state getter into commits", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }));
    act(() => result.current?.setGetState(() => ({ title: "Spec", body: '{"type":"doc"}' })));
    await flushConnect();
    act(() => socket.onopen?.({}));

    // Make a local edit so the session is dirty, then flush.
    act(() => result.current?.doc.getText("body").insert(0, "draft"));
    act(() => result.current?.provider.flushCommit());

    await waitFor(() => {
      const commits = framesOf(socket, "commit") as unknown as { title: string; body: string }[];
      expect(commits.length).toBeGreaterThan(0);
      expect(commits[0]?.title).toBe("Spec");
      expect(commits[0]?.body).toBe('{"type":"doc"}');
    });
  });

  // Guards two regressions: StrictMode's double-invoke permanently killing the session, and the
  // throwaway instance that fix left opening a real socket of its own.
  it("reconnects cleanly through a StrictMode double-invoke instead of dying on the synthetic cleanup", async () => {
    const sockets: FakeSocket[] = [];
    const factory = () => {
      const socket = new FakeSocket();
      sockets.push(socket);
      return socket;
    };

    const { result } = renderHook(
      () => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }),
      { wrapper: StrictMode },
    );
    expect(result.current).not.toBeNull();

    await flushConnect();

    // The throwaway StrictMode instance never opens a real socket; only the surviving instance's deferred connect() runs.
    expect(sockets).toHaveLength(1);

    act(() => sockets[0]?.onopen?.({}));
    expect(result.current?.connected).toBe(true);
  });
});
