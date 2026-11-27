import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

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

describe("useFetchPlayQueue", () => {
  it("never asks about an interview, which has no queue", () => {
    renderHook(() => useFetchPlayQueue("interview", "p-1"), { wrapper });
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("playQueueFollower", () => {
  it("refetches only the queue of the target, the play, and the board of the project a frame names", async () => {
    const client = seeded([
      [["getPlayQueue", "ticket", "t-1"], playQueue()],
      [["getPlayQueue", "ticket", "t-2"], playQueue()],
      [["getPlayQueued", "play-1"], []],
      [["getPlayQueued", "play-2"], []],
      [["getActiveTrails", "ticket", "p-1"], {}],
      [["getActiveTrails", "ticket", "p-2"], {}],
    ]);
    await followFrame(playQueueFollower, "play.queue_updated", queueItem({ status: "skipped" }), client);
    expect(
      [
        ["getPlayQueue", "ticket", "t-1"], ["getPlayQueue", "ticket", "t-2"], ["getPlayQueued", "play-1"], ["getPlayQueued", "play-2"],
        ["getActiveTrails", "ticket", "p-1"], ["getActiveTrails", "ticket", "p-2"],
      ].map((key) => isStale(client, key)),
    ).toEqual([true, false, true, false, true, false]);
  });
});
