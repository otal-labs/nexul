import { Awareness, applyAwarenessUpdate, encodeAwarenessUpdate } from "y-protocols/awareness";
import * as Y from "yjs";

import type { LiveSocket } from "@/api/ws";
import { fromBase64, toBase64 } from "@/lib/collab/binary";
import { parseServerFrame, type InitFrame } from "@/lib/collab/protocol";

const REMOTE = Symbol("collab-remote");

const COMMIT_INTERVAL_MS = 5000;
const MAX_BACKOFF_MS = 30_000;
const BASE_BACKOFF_MS = 500;
const JITTER_MS = 250;

export interface CollabProviderOptions {
  url: string;
  doc: Y.Doc;
  awareness?: Awareness;
  wsFactory?: (url: string) => LiveSocket;
  /** Called with the current converged title/body when a commit is due. */
  getState?: () => { title: string; body: string } | null;
  /** Commit interval; a no-op unless local edits are pending. Default 5s. */
  commitIntervalMs?: number;
  onInit?: (init: InitFrame) => void;
  onStatus?: (status: "connecting" | "connected" | "disconnected") => void;
  onApplyError?: (error: unknown) => void;
  /** A peer renamed the doc (commit relay carried a title). */
  onRemoteTitle?: (title: string) => void;
}

// Owns the Y.Doc<->transport; the server just relays updates, CRDT merge happens client-side.
export class RelayCollabProvider {
  readonly doc: Y.Doc;
  readonly awareness: Awareness;
  readonly options: CollabProviderOptions;

  private socket: LiveSocket | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private commitTimer: ReturnType<typeof setInterval> | null = null;
  private attempt = 0;
  private destroyed = false;
  private connected = false;
  private pending: Uint8Array[] = [];
  private _lastSeq = 0;
  private _hasServerState = false;
  private _dirty = false;

  constructor(options: CollabProviderOptions) {
    this.options = options;
    this.doc = options.doc;
    this.awareness = options.awareness ?? new Awareness(options.doc);
    // Seed an empty local state so setLocalStateField (cursor plugin) never silently no-ops on its first call.
    this.awareness.setLocalState({});

    this.doc.on("update", this.handleDocUpdate);
    this.awareness.on("change", this.handleAwarenessChange);
  }

  get isConnected(): boolean {
    return this.connected;
  }

  /** True once the server has replayed stored state (never seeds then). */
  get hasServerState(): boolean {
    return this._hasServerState;
  }

  /** True while local edits are not yet persisted by a commit. */
  get dirty(): boolean {
    return this._dirty;
  }

  get lastSeq(): number {
    return this._lastSeq;
  }

  connect() {
    if (this.destroyed) return;
    this.options.onStatus?.("connecting");
    const socket = (this.options.wsFactory ?? ((u) => new WebSocket(u) as unknown as LiveSocket))(this.options.url);
    this.socket = socket;
    socket.onopen = () => this.handleOpen();
    socket.onmessage = (ev) => this.handleMessage(ev.data);
    socket.onerror = () => undefined;
    socket.onclose = () => this.scheduleReconnect();
  }

  destroy() {
    this.destroyed = true;
    if (this.timer) clearTimeout(this.timer);
    if (this.commitTimer) clearInterval(this.commitTimer);
    this.timer = null;
    this.commitTimer = null;
    this.connected = false;
    this.socket?.close();
    this.socket = null;
    this.doc.off("update", this.handleDocUpdate);
    this.awareness.off("change", this.handleAwarenessChange);
    this.awareness.destroy();
  }

  /** The title lives outside the Y.Doc, so a title-only edit needs this to trip the commit loop. */
  markDirty() {
    this._dirty = true;
  }

  /** Force a commit of the converged state (Save button / unmount). */
  flushCommit() {
    if (!this.connected || !this._dirty) return;
    this.commit();
  }

  /** "Keep mine" conflict recovery: rebroadcasts local state as an update, making it the shared truth. */
  recoverKeepMine() {
    if (!this.connected) return;
    this.send({ type: "update", update: toBase64(Y.encodeStateAsUpdate(this.doc)) });
    this._dirty = true;
    this.commit();
  }

  private handleOpen() {
    this.attempt = 0;
    this.connected = true;
    this.options.onStatus?.("connected");
    this.send({ type: "hello", client_id: this.doc.clientID });
    // Y.js merges are commutative, so replay order of these queued edits never matters server-side.
    if (this.pending.length > 0) {
      const merged = Y.mergeUpdates(this.pending);
      this.pending = [];
      this.send({ type: "update", update: toBase64(merged) });
      this._dirty = true;
    }
    this.sendAwareness();
    if (this.commitTimer) clearInterval(this.commitTimer);
    this.commitTimer = setInterval(
      () => this.commit(),
      this.options.commitIntervalMs ?? COMMIT_INTERVAL_MS,
    );
  }

  private scheduleReconnect() {
    if (this.destroyed) return;
    this.connected = false;
    this.options.onStatus?.("disconnected");
    const delay =
      Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** this.attempt) + Math.random() * JITTER_MS;
    this.attempt += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      this.connect();
    }, delay);
  }

  private handleMessage(raw: string) {
    let frame;
    try {
      frame = parseServerFrame(raw);
    } catch {
      return; // malformed frame: drop, never kill the session
    }
    switch (frame.type) {
      case "init":
        this.applyReplay(frame);
        break;
      case "update":
        this.applyRemote(frame.payload);
        if (frame.seq > this._lastSeq) this._lastSeq = frame.seq;
        break;
      case "presence":
        applyAwarenessUpdate(this.awareness, fromBase64(frame.payload), REMOTE);
        break;
      case "leave":
        // y-protocols has no removal API; its own semantics say a null state at the known clock deletes it.
        if (this.awareness.meta.has(frame.client_id)) {
          const removal = encodeAwarenessUpdate(
            this.awareness,
            [frame.client_id],
            new Map([[frame.client_id, null]]) as unknown as Map<number, Record<string, unknown>>,
          );
          applyAwarenessUpdate(this.awareness, removal, REMOTE);
        }
        break;
      case "commit":
        // Informational: the server persisted a converged state.
        if (frame.seq > this._lastSeq) this._lastSeq = frame.seq;
        if (frame.title) this.options.onRemoteTitle?.(frame.title);
        break;
    }
  }

  private applyReplay(init: InitFrame) {
    this._lastSeq = init.seq;
    if (init.snapshot) {
      this._hasServerState = true;
      this.applyRemote(init.snapshot.payload);
    }
    if (init.updates.length > 0) {
      this._hasServerState = true;
      for (const u of init.updates) this.applyRemote(u.payload);
    }
    for (const p of init.presence) applyAwarenessUpdate(this.awareness, fromBase64(p.payload), REMOTE);
    this.options.onInit?.(init);
  }

  private applyRemote(payload: string) {
    try {
      Y.applyUpdate(this.doc, fromBase64(payload), REMOTE);
    } catch (error) {
      // Unresolvable payload: surface the conflict fallback instead of silently diverging.
      this.options.onApplyError?.(error);
    }
  }

  private handleDocUpdate = (update: Uint8Array, origin: unknown) => {
    if (origin === REMOTE) return;
    this._dirty = true;
    if (this.connected) {
      this.send({ type: "update", update: toBase64(update) });
      return;
    }
    this.pending.push(update);
  };

  private handleAwarenessChange = (changes: unknown, origin: unknown) => {
    if (origin === REMOTE) return;
    void changes;
    this.sendAwareness();
  };

  private sendAwareness() {
    if (!this.connected) return;
    const state = encodeAwarenessUpdate(this.awareness, [this.doc.clientID]);
    this.send({ type: "presence", client_id: this.doc.clientID, payload: toBase64(state) });
  }

  private commit() {
    if (!this.connected || !this._dirty) return;
    const state = this.options.getState?.();
    if (!state) return;
    this.send({
      type: "commit",
      update: toBase64(Y.encodeStateAsUpdate(this.doc)),
      base_seq: this._lastSeq,
      title: state.title,
      body: state.body,
    });
    this._dirty = false;
  }

  private send(msg: Record<string, unknown>) {
    this.socket?.send?.(JSON.stringify(msg));
  }
}
