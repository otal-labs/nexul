import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { playQueueFollower } from "@/hooks/PlayQueueHooks";
import type { PlayQueue } from "@/models/PlayQueue";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followFrame } from "@/test/followFrame";
import { playQueue, queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket = { id: "t-1", project_id: "p-1", status: "st-backlog", developer: "alice" } as Ticket;

const people = [
  { user_id: "u-alice", login: "alice", display_name: "Alice Moreau", avatar_url: "" },
  { user_id: "u-bob", login: "bob", display_name: "Bob Tanaka", avatar_url: "" },
];

// The viewer, their permissions, and the ticket's queue; no play applies to the ticket's stage.
const mockApi = (queue: () => PlayQueue, viewer = "u-bob", permissions: string[] = []) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { id: viewer, login: people.find((p) => p.user_id === viewer)?.login } } };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/workspaces/ws-1/people") return { data: { people } };
    if (url === "/api/plays/queue") return { data: queue() };
    return { data: [] };
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <PlaysRailSection ticket={ticket} />
    </QueryClientProvider>,
  );
  return client;
};

const offline = queueItem({ reason: "offline" });

beforeEach(() => {
  vi.resetAllMocks();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlaysRailSection auto play signals", () => {
  it("shows a queued run with whom it waits on and why, even when no play applies and the viewer cannot run plays", async () => {
    mockApi(() => playQueue({ items: [offline, queueItem({ id: "q-2", play_label: "Test with AI", person_id: "u-bob", reason: "ticket busy" })] }));
    renderSection();
    expect(await screen.findByRole("heading", { name: "Plays" })).toBeInTheDocument();
    expect(screen.getByText("Fix with AI queued")).toBeInTheDocument();
    expect(await screen.findByText("Alice Moreau · computer offline")).toBeInTheDocument();
    expect(screen.getByText("Bob Tanaka · ticket busy")).toBeInTheDocument();
  });

  it("leaves out runs that already started, skipped or didn't run", async () => {
    mockApi(() => playQueue({ items: [queueItem({ status: "started" }), queueItem({ id: "q-2", status: "skipped", decided_at: "2026-10-10T09:05:00Z" })] }));
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/queue", expect.anything()));
    expect(screen.queryByRole("heading", { name: "Plays" })).not.toBeInTheDocument();
  });

  it("lets the person the run lands on cancel it", async () => {
    mockApi(() => playQueue({ items: [offline] }), "u-alice");
    vi.mocked(api.post).mockResolvedValue({ data: { ...offline, status: "cancelled" } });
    renderSection();
    await userEvent.click(await screen.findByRole("button", { name: "Cancel Fix with AI" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/queue/q-1/cancel");
  });

  it("offers Cancel to someone else only with autoplays:write", async () => {
    mockApi(() => playQueue({ items: [offline] }), "u-bob");
    renderSection();
    expect(await screen.findByText("Fix with AI queued")).toBeInTheDocument();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/me"));
    expect(screen.queryByRole("button", { name: "Cancel Fix with AI" })).not.toBeInTheDocument();

    vi.resetAllMocks();
    mockApi(() => playQueue({ items: [offline] }), "u-bob", ["autoplays:write"]);
    renderSection();
    expect(await screen.findByRole("button", { name: "Cancel Fix with AI" })).toBeInTheDocument();
  });

  it("shows a paused ticket with its runs today, and lets the ticket's developer resume it", async () => {
    mockApi(() => playQueue({ paused: true, auto_runs: 5, paused_until: "2099-01-01T00:00:00Z" }), "u-alice");
    vi.mocked(api.post).mockResolvedValue({ data: playQueue() });
    renderSection();
    expect(await screen.findByText("Auto plays paused")).toBeInTheDocument();
    expect(screen.getByText("runs today", { exact: false })).toHaveTextContent("5 runs today");
    await userEvent.click(await screen.findByRole("button", { name: "Resume" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/queue/resume", { target_type: "ticket", target_id: "t-1" });
    await waitFor(() => expect(screen.queryByText("Auto plays paused")).not.toBeInTheDocument());
  });

  it("offers Resume to no one but the developer and autoplays:write holders", async () => {
    mockApi(() => playQueue({ paused: true, auto_runs: 1, paused_until: "2099-01-01T00:00:00Z" }), "u-bob");
    renderSection();
    expect(await screen.findByText("today", { exact: false })).toHaveTextContent("1 run today");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/me"));
    expect(screen.queryByRole("button", { name: "Resume" })).not.toBeInTheDocument();
  });

  it("follows the queue live: a run queued after the page opened appears without a refresh", async () => {
    let queue = playQueue();
    mockApi(() => queue);
    const client = renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/queue", expect.anything()));
    expect(screen.queryByText("Fix with AI queued")).not.toBeInTheDocument();

    queue = playQueue({ items: [offline] });
    await followFrame(playQueueFollower, "play.queued", offline, client);
    expect(await screen.findByText("Fix with AI queued")).toBeInTheDocument();
  });
});
