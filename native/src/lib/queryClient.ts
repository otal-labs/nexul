import { QueryClient, onlineManager, type Query } from "@tanstack/react-query";

import { retryUnlessClientError } from "@nexul/client-core/queryRetry";

import { ApiError } from "@/api/errors";
import { networkOnlineListener } from "@/lib/onlineStatus";

onlineManager.setEventListener(networkOnlineListener);

// A record's detail query is keyed by its id, or by its key when a link opened it; the cached record names it either way.
export const recordQueries = (key: string, id: string) => ({
  queryKey: [key],
  predicate: (query: Query) => query.queryKey[1] === id || (query.state.data as { id?: unknown } | undefined)?.id === id,
});

// staleTime stays 0: the socket closes in the background, so a query is cached until pushed only by opting in (ADR 0136).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: (failureCount, error) => retryUnlessClientError(failureCount, error instanceof ApiError ? error.status : undefined) },
  },
});
