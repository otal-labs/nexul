import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TrailFacts } from "@/components/play/TrailFacts";
import { TrailHeadline } from "@/components/play/TrailHeadline";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import type { Computer } from "@/models/Pairing";
import type { Trail } from "@/models/Trail";
import { useViewAsStore } from "@/stores/viewAsStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: "u-1",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  computer_id: "pc-1",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "done",
  started_at: "2026-10-29T10:00:00Z",
  ended_at: "2026-10-29T10:05:00Z",
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

const computer = (runner: Computer["runner"]) => ({ id: "pc-1", name: "build-box", runner }) as Computer;

const answerComputers = (computers: Computer[] | "error", memories: object[] | "error" = []) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/pairing/computers" && computers === "error") throw new Error("boom");
    if (url === "/api/pairing/computers") return { data: { computers } };
    if (url === "/api/memories" && memories === "error") throw new Error("boom");
    if (url === "/api/memories") return { data: memories };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-1", login: "sam", display_name: "Sam Okafor", avatar_url: "" }] } };
    return { data: [] };
  });

const renderInDialog = (run: Trail) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <Dialog open>
          <DialogContent>
            <DialogTitle>{run.play_label}</DialogTitle>
            <TrailHeadline trail={run} />
            <TrailFacts trail={run} />
          </DialogContent>
        </Dialog>
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { client, view };
};

afterEach(() => useViewAsStore.setState({ target: null }));

describe("TrailFacts", () => {
  it.each([
    { name: "a connected runner", computers: [computer({ connected: true, reconnecting: false, last_seen: "" })], shows: ["build-box", "Online now"] },
    { name: "a dropped runner coming back", computers: [computer({ connected: false, reconnecting: true, last_seen: "" })], shows: ["build-box", "Reconnecting now"] },
    { name: "a runner that is gone", computers: [computer({ connected: false, reconnecting: false, last_seen: "" })], shows: ["build-box", "Offline now"] },
    { name: "a computer with no runner", computers: [computer(undefined)], shows: ["build-box"], hides: ["Offline now", "Online now"] },
    { name: "a computer the viewer can't see", computers: [], shows: ["A computer you can't see"] },
    { name: "a failed computers request", computers: "error" as const, shows: ["Couldn't load the computer"], hides: ["A computer you can't see", "Loading…"] },
  ])("names the computer and how it stands for $name", async ({ computers, shows, hides = [] }) => {
    answerComputers(computers);
    renderInDialog(trail({}));

    for (const text of shows) expect(await screen.findByText(text)).toBeInTheDocument();
    for (const text of hides) expect(screen.queryByText(text)).not.toBeInTheDocument();
  });

  it("says how a running run's computer stands without the 'now' an ended run adds, and what it passed over", async () => {
    answerComputers([computer({ connected: true, reconnecting: false, last_seen: "" })]);
    renderInDialog(trail({ state: "running", ended_at: null, passed_over: [{ computer_id: "pc-2", name: "laptop", why: "offline", text: "laptop is offline" }] }));

    expect(await screen.findByText("Online")).toBeInTheDocument();
    expect(screen.getByText("Runs on")).toBeInTheDocument();
    expect(screen.getByText("Passed over: laptop is offline")).toBeInTheDocument();
  });

  it("says the computer isn't shown while viewing as another member, without asking for it", () => {
    answerComputers([computer({ connected: true, reconnecting: false, last_seen: "" })]);
    useViewAsStore.setState({ target: { workspaceId: "ws-1", workspaceName: "Northwind", userId: "u-2", name: "Lena", avatarUrl: "" } });
    renderInDialog(trail({}));

    expect(screen.getByText("Not shown while viewing as another member")).toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/pairing/computers");
  });

  it("keeps the computer it named when a later refresh of the list fails", async () => {
    answerComputers([computer({ connected: true, reconnecting: false, last_seen: "" })]);
    const { client } = renderInDialog(trail({}));
    expect(await screen.findByText("build-box")).toBeInTheDocument();

    answerComputers("error");
    await client.refetchQueries();

    expect(screen.getByText("build-box")).toBeInTheDocument();
    expect(screen.getByText("Online now")).toBeInTheDocument();
    expect(screen.queryByText("Couldn't load the computer")).not.toBeInTheDocument();
    expect(screen.queryByText("Loading…")).not.toBeInTheDocument();
  });

  it("says no computer took a run that landed nowhere, and no model either", () => {
    answerComputers([]);
    renderInDialog(trail({ computer_id: "", state: "failed" }));

    expect(screen.getByText("No computer took it")).toBeInTheDocument();
    expect(screen.getByText("None")).toBeInTheDocument();
    expect(screen.queryByText("The computer's default")).not.toBeInTheDocument();
  });

  it("names the picked memories once their list answers, a deleted one as plain text", async () => {
    answerComputers([], [{ id: "m-1", title: "Early return, no else" }]);
    renderInDialog(trail({ selected_memory_ids: ["m-1", "m-gone"] }));

    const memoriesFact = screen.getByText("Memories").parentElement as HTMLElement;
    expect(within(memoriesFact).getByText("Loading…")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "Early return, no else" })).toBeInTheDocument();
    expect(screen.getByText("Deleted memory")).not.toHaveAttribute("href");
  });

  it("says the memories couldn't load when their list fails", async () => {
    answerComputers([], "error");
    renderInDialog(trail({ selected_memory_ids: ["m-1"] }));

    expect(await screen.findByText("Couldn't load the memories")).toBeInTheDocument();
  });

  it("asks for no memories when none were picked", () => {
    answerComputers([]);
    renderInDialog(trail({}));

    expect(screen.getByText("None picked")).toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/memories", expect.anything());
  });
});

describe("TrailModel", () => {
  const providers = [{ id: "codex", driver: "codex", name: "Codex", needs_setup: false, skipped: false, models: [{ slug: "gpt-x", name: "GPT X" }] }];
  const answerProviders = (list: object[] | "error") => {
    answerComputers([computer(undefined)]);
    const base = vi.mocked(api.get).getMockImplementation();
    vi.mocked(api.get).mockImplementation(async (url: string, config?: object) => {
      if (url === "/api/pairing/computers/pc-1/providers" && list === "error") throw new Error("offline");
      if (url === "/api/pairing/computers/pc-1/providers") return { data: { providers: list } };
      return base?.(url, config);
    });
  };

  it("names the model and its provider as the computer lists them", async () => {
    answerProviders(providers);
    renderInDialog(trail({ provider: "codex", model: "gpt-x" }));

    expect(await screen.findByText("Codex")).toBeInTheDocument();
    expect(screen.getByText("GPT X")).toBeInTheDocument();
  });

  it("falls back to the slug's words and provider id while the computer can't say", async () => {
    answerProviders("error");
    renderInDialog(trail({ provider: "codex", model: "gpt-x" }));

    expect(await screen.findByText("GPT X")).toBeInTheDocument();
    expect(screen.getByText("codex")).toBeInTheDocument();
  });

  it("says the computer's default when the run picked none, and asks the computer nothing", () => {
    answerProviders(providers);
    renderInDialog(trail({}));

    expect(screen.getByText("The computer's default")).toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/pairing/computers/pc-1/providers");
  });

  it("asks the computer nothing while viewing as another member", () => {
    answerProviders(providers);
    useViewAsStore.setState({ target: { workspaceId: "ws-1", workspaceName: "Northwind", userId: "u-2", name: "Lena", avatarUrl: "" } });
    renderInDialog(trail({ provider: "codex", model: "gpt-x" }));

    expect(screen.getByText("GPT X")).toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/pairing/computers/pc-1/providers");
  });
});

describe("TrailHeadline", () => {
  it.each([
    { state: "done" as const, reads: "Done in 5m 0s" },
    { state: "failed" as const, reads: "Failed after 5m 0s" },
    { state: "interrupted" as const, reads: "Interrupted after 5m 0s" },
    { state: "waiting" as const, reads: "Waiting for an answer" },
    { state: "starting" as const, reads: "Starting for 5m 0s" },
  ])("reads a $state run as '$reads'", async ({ state, reads }) => {
    answerComputers([]);
    renderInDialog(trail({ state }));

    expect(screen.getByText(reads)).toBeInTheDocument();
  });

  it("names the starter and where they started it, or the automation alone", async () => {
    answerComputers([]);
    renderInDialog(trail({ via: "mcp" }));
    expect(await screen.findByText("Started by Sam Okafor over MCP")).toBeInTheDocument();

    renderInDialog(trail({ id: "tr-2", via: "automation" }));
    expect(screen.getByText("Started by an automation")).toBeInTheDocument();
  });
});
