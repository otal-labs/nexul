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
    { name: "a connected runner", computers: [computer({ connected: true, reconnecting: false, last_seen: "" })], shows: ["build-box", "Online"] },
    { name: "a dropped runner coming back", computers: [computer({ connected: false, reconnecting: true, last_seen: "" })], shows: ["build-box", "Reconnecting"] },
    { name: "a runner that is gone", computers: [computer({ connected: false, reconnecting: false, last_seen: "" })], shows: ["build-box", "Offline"] },
    { name: "a computer with no runner", computers: [computer(undefined)], shows: ["build-box"], hides: ["Offline", "Online"] },
    { name: "a computer the viewer can't see", computers: [], shows: ["A computer you can't see"] },
    { name: "a failed computers request", computers: "error" as const, shows: ["Couldn't load the computer"], hides: ["A computer you can't see", "Loading…"] },
  ])("names the computer and how it stands for $name", async ({ computers, shows, hides = [] }) => {
    answerComputers(computers);
    renderInDialog(trail({}));

    for (const text of shows) expect(await screen.findByText(text)).toBeInTheDocument();
    for (const text of hides) expect(screen.queryByText(text)).not.toBeInTheDocument();
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
    expect(screen.queryByText("Couldn't load the computer")).not.toBeInTheDocument();
  });

  it("says no computer took a run that landed nowhere", () => {
    answerComputers([]);
    renderInDialog(trail({ computer_id: "", state: "failed" }));

    expect(screen.getByText("No computer took it")).toBeInTheDocument();
  });

  it("names the picked memories once their list answers, a deleted one as plain text", async () => {
    answerComputers([], [{ id: "m-1", title: "Early return, no else" }]);
    renderInDialog(trail({ selected_memory_ids: ["m-1", "m-gone"] }));

    const memoriesFact = screen.getByText("Memories").parentElement as HTMLElement;
    expect(within(memoriesFact).getByText("Loading…")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "Early return, no else" })).toBeInTheDocument();
    expect(screen.getByText("Deleted memory")).not.toHaveAttribute("href");
  });

  it("says the memories couldn't load when their list fails, and asks for none when none were picked", async () => {
    answerComputers([], "error");
    renderInDialog(trail({ selected_memory_ids: ["m-1"] }));
    expect(await screen.findByText("Couldn't load the memories")).toBeInTheDocument();

    vi.mocked(api.get).mockClear();
    renderInDialog(trail({ id: "tr-2" }));
    expect(screen.getByText("None picked")).toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/memories", expect.anything());
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
