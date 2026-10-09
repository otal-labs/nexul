const MAX_RETRIES = 3;

// A 4xx is the server's final answer; only network failures (no status) and 5xx are worth another attempt.
export const retryUnlessClientError = (failureCount: number, status: number | undefined): boolean => {
  if (status !== undefined && status >= 400 && status < 500) return false;
  return failureCount < MAX_RETRIES;
};
