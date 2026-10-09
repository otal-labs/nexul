import { QueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";

import { retryUnlessClientError } from "@nexul/client-core/queryRetry";

// staleTime 30s: the socket stays open while signed in, hidden tab included, and pushes every change (ADR 0139).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      // A cast, not axios's isAxiosError: importing that value split the entry into 28 more chunks on first load.
      retry: (failureCount, error) => retryUnlessClientError(failureCount, (error as AxiosError | null)?.response?.status),
    },
  },
});
