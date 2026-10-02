import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { useCallback, useEffect, useEffectEvent, useRef, useState } from "react";

import * as Y from "yjs";

import type { LiveSocket } from "@/api/ws";
import { getDocKey } from "@/hooks/DocHooks";
import { buildCollabURL } from "@/lib/collab/url";
import { RelayCollabProvider } from "@/lib/collab/provider";
import type { Doc } from "@/models/Doc";

// Cool-hue palette only, keyed by y client id, so cursors read as part of the black+blue canvas.
const PRESENCE_COLORS = [
  "#3b82f6",
  "#6366f1",
  "#0ea5e9",
  "#8b5cf6",
  "#06b6d4",
  "#2563eb",
  "#7c3aed",
  "#0284c7",
];

export const colorFor = (clientID: number) => PRESENCE_COLORS[clientID % PRESENCE_COLORS.length] ?? "#6366f1";

export interface CollabUser {
  name: string;
  color: string;
  activity: "viewing" | "editing";
  // Rides awareness state so every participant sees everyone's real avatar; empty means initials.
  avatar?: string;
}

export interface CollabParticipant {
  clientID: number;
  name: string;
  color: string;
  activity: "viewing" | "editing";
  avatar?: string;
}

export interface CollabSession {
  /** Bump to remount the session from scratch (conflict recovery). */
  key: number;
  doc: Y.Doc;
  provider: RelayCollabProvider;
  user: CollabUser;
  /** The server picked this session to load the doc's body into its empty room. */
  seed: boolean;
  connected: boolean;
  participants: CollabParticipant[];
  applyError: boolean;
  /** Registers commit source: title is "" unless this client renamed, to avoid clobbering peers. */
  setGetState: (fn: (() => { title: string; body: string } | null) | null) => void;
  /** Register the peer-rename listener (commit relays carrying a title). */
  setOnRemoteTitle: (fn: ((title: string) => void) | null) => void;
  resetSession: () => void;
  dismissApplyError: () => void;
}

export interface CollabSessionOptions {
  wsFactory?: (url: string) => LiveSocket;
  avatar?: string;
  /** The query a reset reloads before the fresh session starts; a doc room reloads its doc. */
  reloadKey?: QueryKey;
}

// Owns a Y.Doc + provider pair; presence derives from awareness, not a domain event.
// room is the path below /ws/collab: a doc's id, or notes/<message id> for a note's file.
export const useCollabSession = (
  room: string | undefined,
  mode: "edit" | "view",
  name: string,
  token: string | null,
  opts: CollabSessionOptions = {},
): CollabSession | null => {
  const [sessionKey, setSessionKey] = useState(0);
  const [session, setSession] = useState<{ doc: Y.Doc; provider: RelayCollabProvider; user: CollabUser } | null>(
    null,
  );
  const queryClient = useQueryClient();
  const [seededDoc, setSeededDoc] = useState<Y.Doc | null>(null);
  const [connected, setConnected] = useState(false);
  const [applyError, setApplyError] = useState(false);
  const [participants, setParticipants] = useState<CollabParticipant[]>([]);
  const getStateRef = useRef<(() => { title: string; body: string } | null) | null>(null);
  const remoteTitleRef = useRef<((title: string) => void) | null>(null);
  // An effect event, so a caller's inline key array never reconnects the session.
  const reloadKey = useEffectEvent((): QueryKey => opts.reloadKey ?? [getDocKey, room]);

  // One effect (not split) keeps StrictMode's double-invoke setup/cleanup symmetric.
  useEffect(() => {
    // No room or token yet means no connection; the effect re-runs once the token dependency lands.
    if (!room || !token) return;
    const doc = new Y.Doc();
    const user: CollabUser = {
      name: name || "You",
      color: colorFor(doc.clientID),
      activity: mode === "edit" ? "editing" : "viewing",
      ...(opts.avatar ? { avatar: opts.avatar } : {}),
    };
    const provider = new RelayCollabProvider({
      url: buildCollabURL(room, token, mode),
      doc,
      wsFactory: opts.wsFactory ?? ((u) => new WebSocket(u) as unknown as LiveSocket),
      getState: () => getStateRef.current?.() ?? null,
      onRemoteTitle: (title) => remoteTitleRef.current?.(title),
      onSeed: () => setSeededDoc(doc),
      // A server-side write replaced the body: reload it first, so the fresh session seeds the new one, not the old.
      onReset: () => {
        const key = reloadKey();
        void queryClient
          .invalidateQueries({ queryKey: key })
          .then(() => {
            const fresh = queryClient.getQueryData<Partial<Doc>>(key);
            if (typeof fresh?.title === "string") remoteTitleRef.current?.(fresh.title);
          })
          .catch(() => undefined)
          .then(() => setSessionKey((k) => k + 1));
      },
      onStatus: (status) => setConnected(status === "connected"),
      onApplyError: () => setApplyError(true),
    });
    provider.awareness.setLocalStateField("user", user);
    // Deferred a tick so a StrictMode-discarded instance never opens a real socket.
    const connectTimer = setTimeout(() => provider.connect(), 0);
    setSession({ doc, provider, user });

    const refresh = () => {
      const states = provider.awareness.getStates();
      const out: CollabParticipant[] = [];
      for (const [clientID, state] of states) {
        const u = (state as { user?: { name?: string; color?: string; activity?: string; avatar?: string } } | null)
          ?.user;
        if (!u?.name) continue;
        out.push({
          clientID,
          name: u.name,
          color: u.color ?? colorFor(clientID),
          activity: u.activity === "viewing" ? "viewing" : "editing",
          ...(u.avatar ? { avatar: u.avatar } : {}),
        });
      }
      out.sort((a, b) => a.name.localeCompare(b.name));
      // Keeps prior roster identity when unchanged so typing doesn't re-render the whole tree.
      setParticipants((prev) => (JSON.stringify(prev) === JSON.stringify(out) ? prev : out));
    };
    refresh();
    provider.awareness.on("change", refresh);

    return () => {
      clearTimeout(connectTimer);
      provider.awareness.off("change", refresh);
      // Flushes what the 5s commit loop missed — the editor unmounts before cleanup, so refs carry the latest state.
      provider.flushCommit();
      provider.destroy();
      setSession(null);
    };
  }, [room, mode, name, token, sessionKey, opts.wsFactory, opts.avatar, queryClient]);

  const setGetState = useCallback(
    (fn: (() => { title: string; body: string } | null) | null) => {
      getStateRef.current = fn;
    },
    [],
  );

  const setOnRemoteTitle = useCallback((fn: ((title: string) => void) | null) => {
    remoteTitleRef.current = fn;
  }, []);

  const resetSession = useCallback(() => setSessionKey((k) => k + 1), []);
  const dismissApplyError = useCallback(() => setApplyError(false), []);

  if (!session) return null;
  return {
    key: sessionKey,
    doc: session.doc,
    provider: session.provider,
    user: session.user,
    seed: seededDoc === session.doc,
    connected,
    participants,
    applyError,
    setGetState,
    setOnRemoteTitle,
    resetSession,
    dismissApplyError,
  };
};
