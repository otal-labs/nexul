import { QueryObserver } from "@tanstack/react-query";
import { AxiosError, AxiosHeaders } from "axios";
import { afterAll, expect, it, vi } from "vitest";

import { queryClient } from "@/lib/queryClient";

const httpError = (status: number) =>
  new AxiosError("failed", "ERR_BAD_RESPONSE", undefined, undefined, { status, statusText: "", headers: {}, config: { headers: new AxiosHeaders() }, data: null });

// Counts how often a screen's query on the app's client calls a fetch that keeps failing with this error.
const attemptsFor = (error: Error): Promise<number> =>
  new Promise((resolve) => {
    const fetcher = vi.fn().mockRejectedValue(error);
    const observer = new QueryObserver(queryClient, { queryKey: ["retry", Math.random()], queryFn: fetcher, retryDelay: 0 });
    const unsubscribe = observer.subscribe((result) => {
      if (result.status !== "error") return;
      unsubscribe();
      resolve(fetcher.mock.calls.length);
    });
  });

// A missing or forbidden page lands on its not-found screen at once, not after three retries and seven seconds.
it.each([400, 401, 403, 404, 409])("never retries a %i", async (status) => {
  expect(await attemptsFor(httpError(status))).toBe(1);
});

it("retries a 5xx and a network failure three times", async () => {
  expect(await attemptsFor(httpError(503))).toBe(4);
  expect(await attemptsFor(new AxiosError("Network Error", "ERR_NETWORK"))).toBe(4);
});

afterAll(() => queryClient.clear());
