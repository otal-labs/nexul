import { QueryClient, onlineManager, type Query } from "@tanstack/react-query";

import { ApiError } from "@/api/errors";
import { networkOnlineListener } from "@/lib/onlineStatus";

onlineManager.setEventListener(networkOnlineListener);

const maxRetries = 3;

// A 4xx is the server's final answer; only network failures and 5xx are worth another attempt.
const shouldRetry = (failureCount: number, error: unknown) => {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
  return failureCount < maxRetries;
};

// A record's detail query is keyed by its id, or by its key when a link opened it; the cached record names it either way.
export const recordQueries = (key: string, id: string) => ({
  queryKey: [key],
  predicate: (query: Query) => query.queryKey[1] === id || (query.state.data as { id?: unknown } | undefined)?.id === id,
});

export const queryClient = new QueryClient({ defaultOptions: { queries: { retry: shouldRetry } } });
