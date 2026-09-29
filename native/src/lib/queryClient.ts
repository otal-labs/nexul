import { QueryClient, onlineManager } from "@tanstack/react-query";

import { ApiError } from "@/api/client";
import { networkOnlineListener } from "@/lib/onlineStatus";

onlineManager.setEventListener(networkOnlineListener);

const maxRetries = 3;

// A 4xx is the server's final answer; only network failures and 5xx are worth another attempt.
export const shouldRetry = (failureCount: number, error: unknown) => {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
  return failureCount < maxRetries;
};

export const queryClient = new QueryClient({ defaultOptions: { queries: { retry: shouldRetry } } });
