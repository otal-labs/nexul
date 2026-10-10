import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayQueueNowLine } from "@/components/autoplay/PlayQueueNowLine";
import type { Play } from "@/models/Play";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

const play = { id: "play-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket" } as Play;

const people = [
  { user_id: "u-alice", login: "alice", display_name: "Alice Moreau", avatar_url: "" },
  { user_id: "u-bob", login: "bob", display_name: "", avatar_url: "" },
  { user_id: "u-sam", login: "sam", display_name: "", avatar_url: "" },
];

const renderLine = (queued: PlayQueueItem[]) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/plays/queued") return { data: queued };
    return { data: { people } };
  });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <PlayQueueNowLine play={play} />
    </QueryClientProvider>,
  );
};

beforeEach(() => vi.resetAllMocks());

describe("PlayQueueNowLine", () => {
  it("counts what waits for the play and on whom, naming two people and counting the rest", async () => {
    renderLine([
      queueItem({ id: "q-1", reason: "offline" }),
      queueItem({ id: "q-2", target_id: "t-2", reason: "offline" }),
      queueItem({ id: "q-3", person_id: "u-bob" }),
      queueItem({ id: "q-4", person_id: "u-sam", reason: "paused" }),
    ]);
    expect(await screen.findByText(/Right now:/)).toHaveTextContent(
      "Right now: 4 queued · 2 waiting on Alice Moreau (computer offline) · 1 waiting on bob (no free slot) · 1 more waiting",
    );
    expect(api.get).toHaveBeenCalledWith("/api/plays/queued", { params: { play_id: "play-1" } });
  });

  it("says nothing when nothing is queued", async () => {
    renderLine([]);
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/queued", expect.anything()));
    expect(screen.queryByText(/Right now/)).not.toBeInTheDocument();
  });
});
