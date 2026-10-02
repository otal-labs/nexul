import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode, type ReactNode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Awareness, encodeAwarenessUpdate } from "y-protocols/awareness";
import { Doc } from "yjs";

import { api } from "@/api/client";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { useFetchDoc } from "@/hooks/DocHooks";
import type { LiveSocket } from "@/api/ws";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  resolveWSBase: vi.fn(() => "ws://test"),
  errorMessage: vi.fn(),
}));

const withClient = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {children}
  </QueryClientProvider>
);

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
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", null, { wsFactory: factory }), {
      wrapper: withClient,
    });
    expect(result.current).toBeNull();
    await flushConnect();
    expect(socket.send).not.toHaveBeenCalled();
  });

  it("creates the session with a colored presence for the acting user", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }), {
      wrapper: withClient,
    });
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
    const { result } = renderHook(() => useCollabSession("doc-1", "view", "Alice", "token-1", { wsFactory: factory }), {
      wrapper: withClient,
    });
    await flushConnect();
    act(() => socket.onopen?.({}));
    socket.dispatch(JSON.stringify({ type: "presence", from: "bob", client_id: 7, payload: remotePresence() }));

    await waitFor(() => {
      expect(result.current?.participants.some((p) => p.name === "Bob" && p.activity === "viewing")).toBe(true);
    });
  });

  it("flags unresolvable payloads and resets to a fresh session", async () => {
    const { socket, factory } = makeSocket();
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }), {
      wrapper: withClient,
    });
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
    const { result } = renderHook(() => useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory }), {
      wrapper: withClient,
    });
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

  // A reset means a server-side write replaced the body; rejoining before it reloads would seed the old body again.
  it("on a reset, reloads the doc and passes its title on before starting a fresh session", async () => {
    const { socket, factory } = makeSocket();
    vi.mocked(api.get).mockResolvedValueOnce({ data: { id: "doc-1", title: "Plan", body: "{}" } });
    const { result } = renderHook(
      () => {
        useFetchDoc("doc-1");
        return useCollabSession("doc-1", "edit", "Alice", "token-1", { wsFactory: factory });
      },
      { wrapper: withClient },
    );
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1));
    const first = result.current!;
    const onRemoteTitle = vi.fn();
    act(() => first.setOnRemoteTitle(onRemoteTitle));
    await flushConnect();
    act(() => socket.onopen?.({}));

    let reload: (doc: unknown) => void = () => {};
    vi.mocked(api.get).mockReturnValueOnce(new Promise((resolve) => (reload = resolve)));
    act(() => socket.dispatch(JSON.stringify({ type: "reset" })));
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2));
    expect(result.current?.doc).toBe(first.doc);

    await act(async () => reload({ data: { id: "doc-1", title: "Agent title", body: "{}" } }));
    await waitFor(() => expect(result.current?.doc).not.toBe(first.doc));
    expect(onRemoteTitle).toHaveBeenCalledWith("Agent title");
  });

  // A note's room reloads the note's own query on a reset; reloading the doc key would rejoin with the stale file.
  it("joins a note's room by its path and, on a reset, reloads the query the caller names", async () => {
    const urls: string[] = [];
    const { socket } = makeSocket();
    const factory = (url: string) => {
      urls.push(url);
      return socket;
    };
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(
      () =>
        useCollabSession("notes/m-1", "edit", "Alice", "token-1", { wsFactory: factory, reloadKey: ["noteFile", "m-1"] }),
      { wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider> },
    );
    const first = result.current!;
    await flushConnect();
    expect(urls[0]).toBe("ws://test/ws/collab/notes/m-1?mode=edit&token=token-1");
    act(() => socket.onopen?.({}));

    act(() => socket.dispatch(JSON.stringify({ type: "reset" })));
    await waitFor(() => expect(result.current?.doc).not.toBe(first.doc));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["noteFile", "m-1"] });
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
      { wrapper: ({ children }) => <StrictMode>{withClient({ children })}</StrictMode> },
    );
    expect(result.current).not.toBeNull();

    await flushConnect();

    // The throwaway StrictMode instance never opens a real socket; only the surviving instance's deferred connect() runs.
    expect(sockets).toHaveLength(1);

    act(() => sockets[0]?.onopen?.({}));
    expect(result.current?.connected).toBe(true);
  });
});
