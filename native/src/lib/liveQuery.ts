import type { EventPayloads, Topic } from "@nexul/sdk/events";
import { queryOptions, type QueryClient } from "@tanstack/react-query";

import { recordQueries } from "@/lib/queryClient";

// How a frame reaches a query's cache: every entry, the entry its first argument names, the record it names (by id,
// or by the key a link opened it with), or a patch written from the payload with no request.
type Refresh<T extends Topic> =
  | "all"
  | { key: (payload: EventPayloads[T]) => unknown }
  | { record: (payload: EventPayloads[T]) => unknown }
  | { patch: (client: QueryClient, payload: EventPayloads[T]) => void };

type Refreshes = { [T in Topic]?: Refresh<T> };

interface Definition<Args extends unknown[], Data, R extends Refreshes> {
  key: string;
  fetch: (...args: Args) => Promise<Data>;
  refreshes: R & { [K in Exclude<keyof R, Topic>]: never };
  // Cached until a topic refreshes it and refetched only on foreground and reconnect; with no topic nothing ever would.
  untilPushed?: [keyof R] extends [never] ? never : true;
}

type LooseRefresh =
  | "all"
  | { key: (payload: never) => unknown }
  | { record: (payload: never) => unknown }
  | { patch: (client: QueryClient, payload: never) => void };

interface LiveQuery {
  key: string;
  refreshes: Partial<Record<string, LooseRefresh>>;
}

const untilPushedOptions = { staleTime: Infinity, refetchOnWindowFocus: "always", refetchOnReconnect: "always" } as const;

// Filled as each definition's module loads, which is before anything can cache an entry under its key.
const registry = new Map<string, LiveQuery>();

export const liveQueries = (): Iterable<LiveQuery> => registry.values();

export const defineQuery = <Args extends unknown[], Data, const R extends Refreshes>(definition: Definition<Args, Data, R>) => {
  const { key, fetch, refreshes, untilPushed } = definition;
  if (untilPushed && Object.keys(refreshes).length === 0) throw new Error(`${key} is cached until pushed but no topic refreshes it`);
  registry.set(key, { key, refreshes });
  return {
    options: (...args: Args) =>
      queryOptions({ queryKey: [key, ...args], queryFn: () => fetch(...args), ...(untilPushed && untilPushedOptions) }),
  };
};

// A payload that lacks the field falls back to every entry, the same as a frame that names nothing.
const pick = (field: (payload: never) => unknown, payload: unknown): string | undefined => {
  try {
    const value = field(payload as never);
    return typeof value === "string" ? value : undefined;
  } catch {
    return undefined;
  }
};

export const applyRefresh = (client: QueryClient, key: string, refresh: LooseRefresh, payload: unknown) => {
  if (refresh === "all") return void client.invalidateQueries({ queryKey: [key] });
  if ("patch" in refresh) return refresh.patch(client, payload as never);
  if ("key" in refresh) {
    const scope = pick(refresh.key, payload);
    return void client.invalidateQueries({ queryKey: scope ? [key, scope] : [key] });
  }
  const id = pick(refresh.record, payload);
  void client.invalidateQueries(id ? recordQueries(key, id) : { queryKey: [key] });
};
