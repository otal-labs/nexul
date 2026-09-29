import { ApiError } from "@/api/client";
import { shouldRetry } from "@/lib/queryClient";

jest.mock("@/lib/onlineStatus", () => ({ networkOnlineListener: jest.fn() }));

describe("shouldRetry", () => {
  test.each([400, 401, 404, 409, 429])("never retries a %i", (status) => {
    expect(shouldRetry(0, new ApiError(status, null, "failed"))).toBe(false);
  });

  test("retries a 5xx and a network failure, up to three times", () => {
    expect(shouldRetry(0, new ApiError(503, null, "failed"))).toBe(true);
    expect(shouldRetry(2, new TypeError("Network request failed"))).toBe(true);
    expect(shouldRetry(3, new TypeError("Network request failed"))).toBe(false);
  });
});
