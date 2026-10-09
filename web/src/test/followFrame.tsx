import { QueryClient } from "@tanstack/react-query";
import type { Location } from "react-router";
import { vi } from "vitest";

import type { LiveFollower } from "@/lib/live";

const home: Location = { pathname: "/", search: "", hash: "", state: null, key: "default" };

// Hands one frame to a follower the way the socket does, and resolves once the follower's work has settled.
export const followFrame = async (follower: LiveFollower, topic: string, payload: unknown, client: QueryClient, location: Location = home) => {
  const follow = follower[topic];
  if (!follow) throw new Error(`the follower does not follow ${topic}`);
  const navigate = vi.fn();
  await follow(payload as never, { client, navigate, location });
  return { navigate };
};

// Whether a frame left a cached query stale, so its next read goes back to the server.
export const isStale = (client: QueryClient, queryKey: unknown[]) => client.getQueryState(queryKey)?.isInvalidated ?? false;

// A client holding the given queries' data, as the views that read them would have left it.
export const seeded = (entries: [unknown[], unknown][]) => {
  const client = new QueryClient();
  for (const [key, data] of entries) client.setQueryData(key, data);
  return client;
};
