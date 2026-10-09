import { QueryClient, onlineManager } from "@tanstack/react-query";

import { ApiError } from "@/api/errors";
import { networkOnlineListener } from "@/lib/onlineStatus";

onlineManager.setEventListener(networkOnlineListener);

const maxRetries = 3;

// A 4xx is the server's final answer; only network failures and 5xx are worth another attempt.
const shouldRetry = (failureCount: number, error: unknown) => {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
  return failureCount < maxRetries;
};

// Reference data list rows read as they mount: only the socket, a foreground or a reconnect refetches it, never a mount.
export const referenceDataOptions = { staleTime: Infinity, refetchOnWindowFocus: "always", refetchOnReconnect: "always" } as const;

export const queryClient = new QueryClient({ defaultOptions: { queries: { retry: shouldRetry } } });
