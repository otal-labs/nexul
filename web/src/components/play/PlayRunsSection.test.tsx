import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayRunsSection } from "@/components/play/PlayRunsSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Play } from "@/models/Play";
import type { PlayRun } from "@/models/Trail";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (error: unknown) => (error as Error).message,
}));

const play = { id: "pl-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket" } as Play;

const run = (id: string, overrides: Partial<PlayRun> = {}): PlayRun => ({
  id,
  workspace_id: "ws-1",
  play_id: "pl-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: `t-${id}`,
  project_id: "p-1",
  conversation_id: "",
  starter_id: "u-me",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "done",
  started_at: "2026-10-29T10:00:00Z",
  ended_at: null,
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
  target_key: `ATL-${id}`,
  target_title: `Ticket ${id}`,
  auto_play_id: "",
  ...overrides,
});

const sevenWaiting = Array.from({ length: 7 }, (_, i) => run(`w${i}`, { state: "waiting", starter_id: i === 0 ? "u-me" : "u-sam" }));
let waiting: PlayRun[] = [];
let recent: PlayRun[] = [];

const serve = () =>
  vi.mocked(api.get).mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    const params = config?.params as { state?: string; limit: number } | undefined;
    if (url === "/api/plays/runs" && params?.state === "waiting") return { data: { items: waiting.slice(0, params.limit), total: waiting.length } };
    if (url === "/api/plays/runs") return { data: { items: recent.slice(0, params?.limit), total: recent.length } };
    if (url === "/api/auth/me") return { data: { user: { id: "u-me" } } };
    if (url === "/api/workspaces/ws-1/people") {
      return { data: { people: [{ user_id: "u-me", login: "alice", display_name: "Alice", avatar_url: "" }, { user_id: "u-sam", login: "sam", display_name: "Sam", avatar_url: "" }] } };
    }
    throw new Error(`unexpected GET ${url}`);
  });

const renderSection = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <PlayRunsSection play={play} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  waiting = [];
  recent = [];
  serve();
});

describe("PlayRunsSection", () => {
  it("lists five runs waiting on an answer, whose answer each waits for, and Show all reads the rest", async () => {
    waiting = sevenWaiting;
    recent = sevenWaiting;
    const user = userEvent.setup();
    renderSection();

    const needs = (await screen.findByText("Needs answer")).parentElement!;
    expect(within(needs).getByText("7")).toBeInTheDocument();
    const rows = () => within(needs).getAllByRole("button", { name: /Ticket w/ });
    expect(rows()).toHaveLength(5);
    expect(await within(rows()[0]!).findByText("Needs your answer")).toBeInTheDocument();
    expect(await within(rows()[1]!).findByText("Needs Sam's answer")).toBeInTheDocument();

    await user.click(within(needs).getByRole("button", { name: "Show all 7" }));
    expect(await within(needs).findByRole("button", { name: /Ticket w6/ })).toBeInTheDocument();
    expect(within(needs).queryByRole("button", { name: /Show all/ })).not.toBeInTheDocument();
  });

  it("reads ten recent runs at a time, says who or what started each, and says when there are none", async () => {
    const user = userEvent.setup();
    recent = Array.from({ length: 12 }, (_, i) => run(`r${i}`, i === 0 ? { auto_play_id: "ap-1" } : i === 1 ? { via: "automation" } : {}));
    const { unmount } = renderSection();

    const list = (await screen.findByText("Recent")).parentElement!;
    expect(within(list).getAllByRole("button", { name: /Ticket r/ })).toHaveLength(10);
    expect(within(list).getByRole("button", { name: /Ticket r0(?!\d)/ })).toHaveTextContent("Auto");
    expect(within(list).getByRole("button", { name: /Ticket r1(?!\d)/ })).toHaveTextContent("Automation");
    await user.click(within(list).getByRole("button", { name: "Show more" }));
    expect(await within(list).findByRole("button", { name: /Ticket r11/ })).toBeInTheDocument();
    expect(within(list).queryByRole("button", { name: "Show more" })).not.toBeInTheDocument();
    unmount();

    recent = [];
    renderSection();
    expect(await screen.findByText("No runs yet")).toBeInTheDocument();
    expect(screen.queryByText("Needs answer")).not.toBeInTheDocument();
  });
});
