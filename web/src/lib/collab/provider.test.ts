import { describe, expect, it, vi } from "vitest";

import * as Y from "yjs";
import { Awareness, encodeAwarenessUpdate } from "y-protocols/awareness";

import type { LiveSocket } from "@/api/ws";
import { fromBase64, toBase64 } from "@/lib/collab/binary";
import { RelayCollabProvider } from "@/lib/collab/provider";

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  sent: string[] = [];

  send = vi.fn((raw: string) => {
    this.sent.push(raw);
  });

  close = vi.fn(() => {
    this.onclose?.({});
  });

  dispatch(raw: string) {
    this.onmessage?.({ data: raw });
  }

  frames<T>(): T[] {
    return this.sent.map((raw) => JSON.parse(raw) as T);
  }

  framesOf(type: string) {
    return this.frames<{ type: string }>().filter((f) => f.type === type);
  }
}

const setup = (overrides: Partial<ConstructorParameters<typeof RelayCollabProvider>[0]> = {}) => {
  const doc = new Y.Doc();
  const socket = new FakeSocket();
  const provider = new RelayCollabProvider({
    url: "ws://test/collab/doc-1?mode=edit&token=t",
    doc,
    wsFactory: () => socket,
    ...overrides,
  });
  provider.connect();
  socket.onopen?.({});
  return { doc, socket, provider };
};

const encode = (doc: Y.Doc) => toBase64(Y.encodeStateAsUpdate(doc));

describe("RelayCollabProvider", () => {
  it("announces its client id on connect", () => {
    const { doc, socket } = setup();
    const hello = socket.framesOf("hello")[0] as unknown as { client_id: number };
    expect(hello.client_id).toBe(doc.clientID);
  });

  it("applies the replayed snapshot and increments in order", () => {
    const server = new Y.Doc();
    const fragment = server.getText("body");
    fragment.insert(0, "hello");
    const snapshot = encode(server);
    // A sequential delta: bob's insert built on alice's state.
    const inc = new Y.Doc();
    Y.applyUpdate(inc, fromBase64(snapshot));
    inc.getText("body").insert(5, " world");
    const increment = encode(inc);

    const onInit = vi.fn();
    const { doc, socket, provider } = setup({ onInit });
    socket.dispatch(
      JSON.stringify({
        type: "init",
        seq: 7,
        snapshot: { seq: 5, kind: "snapshot", actor_id: "alice", payload: snapshot },
        updates: [{ seq: 6, kind: "update", actor_id: "bob", payload: increment }],
        presence: [],
      }),
    );

    expect(doc.getText("body").toString()).toBe("hello world");
    expect(provider.hasServerState).toBe(true);
    expect(provider.lastSeq).toBe(7);
    expect(onInit).toHaveBeenCalledOnce();
  });

  it("marks hasServerState when the replay is empty (fresh doc)", () => {
    const { socket, provider } = setup();
    socket.dispatch(JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }));
    expect(provider.hasServerState).toBe(false);
    expect(provider.lastSeq).toBe(0);
  });

  it("relays local edits immediately and rebroadcasts after reconnect", () => {
    const { doc, socket } = setup();
    doc.getText("body").insert(0, "offline edit");
    expect(socket.framesOf("update")).toHaveLength(1);
    expect(doc.getText("body").toString()).toBe("offline edit");

    socket.onclose?.({});
    doc.getText("body").insert(12, " while down");
    expect(socket.framesOf("update")).toHaveLength(1); // buffered, not sent

    socket.onopen?.({});
    const updates = socket.framesOf("update");
    expect(updates).toHaveLength(2);
    // The reconnect flush is a delta against the session state the server already has (the first edit), so a fresh doc must apply both frames in order to reconstruct the content.
    const merged = new Y.Doc();
    const first = updates[0] as unknown as { update: string };
    const replay = updates[1] as unknown as { update: string };
    Y.applyUpdate(merged, fromBase64(first.update));
    Y.applyUpdate(merged, fromBase64(replay.update));
    expect(merged.getText("body").toString()).toBe("offline edit while down");
  });

  it("applies relayed remote updates and tracks base_seq", () => {
    const { doc, socket, provider } = setup();
    const other = new Y.Doc();
    other.getText("body").insert(0, "theirs");
    socket.dispatch(JSON.stringify({ type: "update", from: "bob", seq: 3, payload: encode(other) }));
    expect(doc.getText("body").toString()).toBe("theirs");
    expect(provider.lastSeq).toBe(3);
  });

  it("commits converged state when dirty, with the applied seq as base", () => {
    const { doc, socket, provider } = setup({
      getState: () => ({ title: "Spec", body: '{"type":"doc"}' }),
      commitIntervalMs: 50,
    });
    doc.getText("body").insert(0, "draft");
    socket.dispatch(JSON.stringify({ type: "update", from: "bob", seq: 4, payload: encode(new Y.Doc()) }));

    vi.waitFor(() => {
      const commits = socket.framesOf("commit") as unknown as { base_seq: number; title: string; body: string; update: string }[];
      expect(commits.length).toBeGreaterThan(0);
      expect(commits[0]).toMatchObject({ base_seq: 4, title: "Spec", body: '{"type":"doc"}' });
      expect(provider.dirty).toBe(false);
    });
  });

  it("skips commits while clean and flushes on demand", () => {
    const { socket, provider } = setup({ getState: () => ({ title: "T", body: "{}" }), commitIntervalMs: 20 });
    socket.dispatch(JSON.stringify({ type: "init", seq: 1, snapshot: null, updates: [], presence: [] }));
    vi.waitFor(() => expect(socket.framesOf("commit")).toHaveLength(0));
    provider.flushCommit();
    expect(socket.framesOf("commit")).toHaveLength(0);

    provider.doc.getText("x").insert(0, "edit");
    provider.flushCommit();
    expect(socket.framesOf("commit")).toHaveLength(1);
  });

  it("markDirty makes a title-only edit commit — the title lives outside the Y.Doc", () => {
    const { socket, provider } = setup({ getState: () => ({ title: "Renamed", body: "{}" }), commitIntervalMs: 20 });
    socket.dispatch(JSON.stringify({ type: "init", seq: 1, snapshot: null, updates: [], presence: [] }));
    provider.flushCommit();
    expect(socket.framesOf("commit")).toHaveLength(0);

    provider.markDirty();
    provider.flushCommit();
    const commits = socket.framesOf("commit") as unknown as { title: string }[];
    expect(commits).toHaveLength(1);
    expect(commits[0]).toMatchObject({ title: "Renamed" });
  });

  it("relays awareness locally and applies remote presence and leave", () => {
    const doc = new Y.Doc();
    const socket = new FakeSocket();
    const awareness = new Awareness(doc);
    const provider = new RelayCollabProvider({
      url: "ws://test/collab/doc-1?mode=edit&token=t",
      doc,
      awareness,
      wsFactory: () => socket,
    });
    provider.connect();
    socket.onopen?.({});
    awareness.setLocalStateField("user", { name: "Alice" });

    const sent = socket.framesOf("presence")[0] as unknown as { client_id: number; payload: string };
    expect(sent.client_id).toBe(doc.clientID);

    const remoteAwareness = new Awareness(new Y.Doc());
    remoteAwareness.clientID = 42;
    // Set twice: y-protocols skips a client's very first state (clock 0), so real presence always rides clock >= 1.
    remoteAwareness.setLocalState({ user: { name: "Bob" } });
    remoteAwareness.setLocalState({ user: { name: "Bob" } });
    socket.dispatch(
      JSON.stringify({
        type: "presence",
        from: "bob",
        client_id: 42,
        payload: toBase64(encodeAwarenessUpdate(remoteAwareness, [42])),
      }),
    );
    expect(awareness.getStates().has(42)).toBe(true);

    socket.dispatch(JSON.stringify({ type: "leave", client_id: 42 }));
    expect(awareness.getStates().has(42)).toBe(false);
  });

  it("drops malformed frames without dying", () => {
    const { socket } = setup();
    expect(() => socket.dispatch("not json")).not.toThrow();
    socket.dispatch(JSON.stringify({ type: "nope" }));
  });

  it("surfaces apply errors for unresolvable payloads", () => {
    const onApplyError = vi.fn();
    const { socket } = setup({ onApplyError });
    socket.dispatch(
      JSON.stringify({ type: "update", from: "bob", seq: 1, payload: toBase64(new Uint8Array([9, 9, 9])) }),
    );
    expect(onApplyError).toHaveBeenCalled();
  });

  it("reconnects with backoff after the socket closes", () => {
    vi.useFakeTimers();
    const factory = vi.fn(() => new FakeSocket());
    const doc = new Y.Doc();
    const provider = new RelayCollabProvider({ url: "ws://test", doc, wsFactory: factory });
    provider.connect();
    const first = factory.mock.results[0]?.value as FakeSocket;
    first.onopen?.({});
    first.onclose?.({});
    vi.advanceTimersByTime(1000);
    expect(factory).toHaveBeenCalledTimes(2);
    provider.destroy();
    vi.useRealTimers();
  });
});
