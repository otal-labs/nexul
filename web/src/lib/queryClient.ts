import { QueryClient } from "@tanstack/react-query";
import { isAxiosError } from "axios";

import { retryUnlessClientError } from "@nexul/client-core/queryRetry";

// staleTime 30s: the socket stays open while signed in, hidden tab included, and pushes every change (ADR 0139).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => retryUnlessClientError(failureCount, isAxiosError(error) ? error.response?.status : undefined),
    },
  },
});
