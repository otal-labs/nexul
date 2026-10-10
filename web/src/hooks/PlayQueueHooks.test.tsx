import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { playQueueFollower, useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";
import { playQueue, queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

let client: QueryClient;
const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockReset();
});

afterEach(() => vi.useRealTimers());

describe("useFetchPlayQueue", () => {
  it("looks again when the daily cap frees a paused ticket, since no frame says so", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true, now: new Date("2026-10-10T09:00:00Z") });
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: playQueue({ paused: true, auto_runs: 5, paused_until: "2026-10-10T10:00:00Z" }) })
      .mockResolvedValueOnce({ data: playQueue({ paused: false, auto_runs: 4 }) });
    const { result } = renderHook(() => useFetchPlayQueue("ticket", "t-1"), { wrapper });
    await waitFor(() => expect(result.current.data?.paused).toBe(true));

    await vi.advanceTimersByTimeAsync(59 * 60 * 1000);
    expect(api.get).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(2 * 60 * 1000);
    await waitFor(() => expect(result.current.data?.paused).toBe(false));
    expect(api.get).toHaveBeenCalledTimes(2);
  });

  it("never asks about an interview, which has no queue", () => {
    renderHook(() => useFetchPlayQueue("interview", "p-1"), { wrapper });
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("playQueueFollower", () => {
  it("refetches only the queue of the target and the play a frame names", async () => {
    const client = seeded([
      [["getPlayQueue", "ticket", "t-1"], playQueue()],
      [["getPlayQueue", "ticket", "t-2"], playQueue()],
      [["getPlayQueued", "play-1"], []],
      [["getPlayQueued", "play-2"], []],
    ]);
    await followFrame(playQueueFollower, "play.queue_updated", queueItem({ status: "skipped" }), client);
    expect(
      [["getPlayQueue", "ticket", "t-1"], ["getPlayQueue", "ticket", "t-2"], ["getPlayQueued", "play-1"], ["getPlayQueued", "play-2"]].map((key) => isStale(client, key)),
    ).toEqual([true, false, true, false]);

    await followFrame(playQueueFollower, "play.queue_resumed", { target_type: "ticket", target_id: "t-2", resumed_by: "u-1" }, client);
    expect(isStale(client, ["getPlayQueue", "ticket", "t-2"])).toBe(true);
  });
});
