import type { QueryClient, QueryKey } from "@tanstack/react-query";
import type { Location, NavigateFunction } from "react-router";

import { resolveWSBase } from "@/api/client";

export const buildLiveURL = (token: string) => {
  const base = import.meta.env.VITE_WS_URL || `${resolveWSBase()}/ws/events`;
  return `${base}?token=${encodeURIComponent(token)}`;
};

// Same session-token query auth as the events socket; the tail is how many past lines arrive before the follow.
export const buildLogsURL = (token: string, stackId: string, service: string, tail: number) =>
  `${resolveWSBase()}/ws/stacks/${stackId}/services/${encodeURIComponent(service)}/logs?tail=${tail}&token=${encodeURIComponent(token)}`;

// What a follower gets beside each frame: the query cache, and the router as it stands when the frame lands.
export interface Live {
  client: QueryClient;
  navigate: NavigateFunction;
  location: Location;
}

// One domain's live follower: per topic it follows, what a frame does to that domain's cache (ADR 0134).
export type LiveFollower = Readonly<Record<string, (payload: never, live: Live) => unknown>>;

// The same follow for several topics whose frames share a payload.
export const followEach = <P>(topics: readonly string[], follow: (payload: P, live: Live) => unknown): LiveFollower =>
  Object.fromEntries(topics.map((topic) => [topic, follow]));

interface Row {
  id: string;
  position: number;
}

// Puts a changed row in place in one cached list; a row that changed its order, or is new, refetches the list.
export const replaceRow = <T extends Row>(client: QueryClient, queryKey: QueryKey, row: T, sameOrder = (a: T, b: T) => a.position === b.position) => {
  const list = client.getQueryData<T[]>(queryKey);
  if (!list) return;
  const old = list.find((r) => r.id === row.id);
  if (!old || !sameOrder(old, row)) return client.invalidateQueries({ queryKey, exact: true });
  client.setQueryData<T[]>(queryKey, list.map((r) => (r.id === row.id ? { ...r, ...row } : r)));
};

export const dropRow = (client: QueryClient, queryKey: QueryKey, id: string) =>
  client.setQueryData<{ id: string }[]>(queryKey, (list) => list?.filter((r) => r.id !== id));

// Refetches the cached queries under a key whose data holds what the frame names; the rest keep their data.
export const refetchHolding = <T>(client: QueryClient, queryKey: QueryKey, holds: (data: T) => boolean) =>
  client.invalidateQueries({ queryKey, predicate: ({ state }) => state.data !== undefined && holds(state.data as T) });
