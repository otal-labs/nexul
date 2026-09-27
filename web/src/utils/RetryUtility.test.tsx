import { afterEach, describe, expect, it, vi } from "vitest";

import { retry } from "@/utils/RetryUtility";

describe("retry", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("resolves as soon as the task does", async () => {
    const task = vi.fn().mockResolvedValue("ok");
    await expect(retry(3, 1000, task)).resolves.toBe("ok");
    expect(task).toHaveBeenCalledTimes(1);
  });

  it("tries again after the delay until the task succeeds", async () => {
    vi.useFakeTimers();
    const task = vi.fn().mockRejectedValueOnce(new Error("not yet")).mockResolvedValueOnce("ok");
    const result = retry(3, 1000, task);
    await vi.advanceTimersByTimeAsync(1000);
    await expect(result).resolves.toBe("ok");
    expect(task).toHaveBeenCalledTimes(2);
  });

  it("rethrows the last failure once the attempts run out", async () => {
    vi.useFakeTimers();
    const task = vi.fn().mockRejectedValue(new Error("still down"));
    const result = retry(2, 1000, task);
    const settled = expect(result).rejects.toThrow("still down");
    await vi.advanceTimersByTimeAsync(1000);
    await settled;
    expect(task).toHaveBeenCalledTimes(2);
  });
});
