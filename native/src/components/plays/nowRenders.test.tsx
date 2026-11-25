import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react-native";

import { api } from "@/api/client";
import { NowSection } from "@/components/plays/NowSection";
import { dispatch } from "@/hooks/useLiveEvents";
import type { WorkspacePlay } from "@/models/Play";

jest.mock("expo-secure-store", () => ({ getItem: () => null, setItem: jest.fn(), deleteItemAsync: jest.fn() }));
jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));
jest.mock("expo-router", () => ({ useRouter: () => ({ push: jest.fn() }) }));

// Each row renders its start time once per render, so the count per start time is that row's render count.
const mockRenders = new Map<string, number>();
jest.mock("@/components/RelativeTime", () => ({
  RelativeTime: ({ iso }: { iso: string }) => {
    mockRenders.set(iso, (mockRenders.get(iso) ?? 0) + 1);
    return iso;
  },
}));

notifyManager.setScheduler((callback) => callback());

const RUNS = 30;
const startedAt = (i: number) => `2026-10-01T10:${String(i).padStart(2, "0")}:00Z`;
const run = (i: number) => ({
  id: `r-${i}`, workspace_id: "ws-1", play_id: "pl-1", play_label: "Fix with AI", target_type: "ticket", target_id: `t-${i}`, target_key: `WEB-${i}`,
  target_title: `Ticket ${i}`, auto_play_id: "", project_id: "p-1", starter_id: "alice", custom_instructions: "", state: "running",
  started_at: startedAt(i), ended_at: null, last_error: "", activity: [],
});

const play = { id: "pl-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket", running: RUNS, waiting: 0, auto_plays_on: 0 } as WorkspacePlay;

const frame = (i: number, summary: string) => ({
  topic: "play.run",
  type: "direct",
  payload: {
    trail_id: `r-${i}`, play_id: "pl-1", target_type: "ticket", target_id: `t-${i}`, state: "running", question: null, ended_at: null, last_error: "",
    activity: { kind: "tool_call", call_id: `c-${summary}`, tool: "Bash", summary, at: "2026-10-01T11:00:00Z" },
    starter_id: "alice", project_id: "p-1", workspace_id: "ws-1",
  },
});

test("a step of one run re-renders that run's row in the Now list and no other", async () => {
  jest.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/plays/runs?play_id=pl-1&limit=20") return { items: Array.from({ length: 20 }, (_, i) => run(i)), total: RUNS };
    throw new Error(`unexpected GET ${url}`);
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <NowSection plays={[play]} />
    </QueryClientProvider>,
  );
  await screen.findByText("WEB-3 · Fix with AI");
  // A run's first frame moves its state from unknown, which refetches its play's pages; the steps after it patch.
  await act(async () => dispatch(client)(frame(3, "go vet ./...")));
  await waitFor(() => expect(client.isFetching()).toBe(0));
  mockRenders.clear();

  await act(async () => dispatch(client)(frame(3, "go test ./...")));

  expect(screen.getByText("go test ./...")).toBeTruthy();
  expect(Object.fromEntries(mockRenders)).toEqual({ [startedAt(3)]: 1 });
});
