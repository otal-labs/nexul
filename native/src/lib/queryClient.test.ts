import { ApiError } from "@/api/errors";
import { queryClient } from "@/lib/queryClient";

jest.mock("@/lib/onlineStatus", () => ({ networkOnlineListener: jest.fn() }));

// Counts how often the app's query client calls a fetch that keeps failing with this error.
const attemptsFor = async (error: Error): Promise<number> => {
  const fetcher = jest.fn().mockRejectedValue(error);
  await queryClient.fetchQuery({ queryKey: ["retry", Math.random()], queryFn: fetcher, retryDelay: 0 }).catch(() => undefined);
  return fetcher.mock.calls.length;
};

test.each([400, 401, 404, 409, 429])("a %i is never retried", async (status) => {
  expect(await attemptsFor(new ApiError(status, null, "failed"))).toBe(1);
});

test("a 5xx and a network failure are retried three times", async () => {
  expect(await attemptsFor(new ApiError(503, null, "failed"))).toBe(4);
  expect(await attemptsFor(new TypeError("Network request failed"))).toBe(4);
});

afterAll(() => queryClient.clear());
