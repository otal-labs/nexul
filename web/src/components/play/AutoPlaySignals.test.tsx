import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import { getProjectLinksKey } from "@/hooks/PairingProjectHooks";
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

const noDefault = queueItem({ reason: "no_default_computer" });

// Alice's two computers: the workstation answers with its T3 projects, the laptop is offline.
const mockComputers = (queue: () => PlayQueue, viewer: string, permissions: string[] = []) => {
  const base = mockApi(queue, viewer, permissions).getMockImplementation()!;
  vi.mocked(api.get).mockImplementation(async (url: string, config?: unknown) => {
    if (url === "/api/pairing/computers") return { data: { computers: [{ id: "c-desk", name: "Workstation" }, { id: "c-laptop", name: "Laptop" }] } };
    if (url === "/api/pairing/presence") return { data: { computers: { "c-desk": "connected", "c-laptop": "offline" } } };
    if (url === "/api/pairing/computers/c-desk/projects") return { data: { projects: [{ id: "t3-web", title: "web", path: "/src/web" }] } };
    return base(url, config as never);
  });
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PlaysRailSection ticket={ticket} />
      </MemoryRouter>
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

  it("links the person a run waits on to the fix for their computer setup", async () => {
    mockApi(() => playQueue({ items: [queueItem({ reason: "expired_token" })] }), "u-alice");
    renderSection();
    expect(await screen.findByRole("link", { name: "Pair it again" })).toHaveAttribute("href", "/settings/pairing");
  });

  it("says only what the gap is to anyone else, since the fix is in the run's person's own settings", async () => {
    mockApi(() => playQueue({ items: [queueItem({ reason: "expired_token" })] }), "u-bob");
    renderSection();
    expect(await screen.findByText(/the paired computer's session expired/)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Pair it again" })).not.toBeInTheDocument();
  });

  it("offers a pick of computer only to the person the run lands on, and only when no computer is the default", async () => {
    mockApi(() => playQueue({ items: [noDefault] }), "u-bob", ["autoplays:write"]);
    renderSection();
    expect(await screen.findByRole("button", { name: "Cancel Fix with AI" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Pick a computer" })).not.toBeInTheDocument();
  });

  it("offers no pick when the run waits on another gap", async () => {
    mockApi(() => playQueue({ items: [queueItem({ reason: "unpaired" })] }), "u-alice");
    renderSection();
    expect(await screen.findByRole("button", { name: "Cancel Fix with AI" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Pick a computer" })).not.toBeInTheDocument();
  });

  it("offers its person a change of computer while a picked run waits, and nobody else", async () => {
    const picked = queueItem({ reason: "offline", computer_id: "c-laptop" });
    mockComputers(() => playQueue({ items: [picked] }), "u-alice");
    vi.mocked(api.post).mockResolvedValue({ data: { ...picked, reason: "", computer_id: "c-desk" } });
    renderSection();
    expect(await screen.findByText("Alice Moreau · computer offline")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Pick a computer" })).not.toBeInTheDocument();
    await userEvent.click(await screen.findByRole("button", { name: "Change computer" }));
    await userEvent.click(await screen.findByRole("menuitem", { name: "web" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/queue/q-1/computer", { computer_id: "c-desk", harness_project_id: "t3-web" });
  });

  it.each([
    ["another person", queueItem({ reason: "offline", computer_id: "c-laptop" }), "u-bob"],
    ["a run already starting", queueItem({ status: "dispatching", computer_id: "c-laptop" }), "u-alice"],
  ])("offers no change of computer to %s", async (_, item, viewer) => {
    mockApi(() => playQueue({ items: [item] }), viewer, ["autoplays:write"]);
    renderSection();
    expect(await screen.findByText("Fix with AI queued")).toBeInTheDocument();
    expect(await screen.findByText(/^Alice Moreau · /)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Change computer" })).not.toBeInTheDocument();
  });

  it("runs the queued run on the computer and T3 project its person picks, and refreshes their links", async () => {
    mockComputers(() => playQueue({ items: [noDefault] }), "u-alice");
    vi.mocked(api.post).mockResolvedValue({ data: { ...noDefault, reason: "", computer_id: "c-desk" } });
    const client = renderSection();
    client.setQueryData([getProjectLinksKey], []);
    await userEvent.click(await screen.findByRole("button", { name: "Pick a computer" }));
    expect(await screen.findByText("Laptop")).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Offline" })).toHaveAttribute("aria-disabled", "true");
    await userEvent.click(await screen.findByRole("menuitem", { name: "web" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/queue/q-1/computer", { computer_id: "c-desk", harness_project_id: "t3-web" });
    await waitFor(() => expect(client.getQueryState([getProjectLinksKey])?.isInvalidated).toBe(true));
  });
});
