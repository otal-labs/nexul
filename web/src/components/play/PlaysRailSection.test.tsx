import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import type { ListedPlay } from "@/models/Play";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket = { id: "t-1", project_id: "p-1", status: "st-backlog" } as Ticket;

const play = { id: "play-1", label: "Fix with AI", type: "ticket", description: "Fix it", show_when_stage: "backlog", can_run: true } as ListedPlay;

const mockApi = (plays: ListedPlay[], permissions = ["plays:run"], resolve: object = {}, runs: object[] = []) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/pairing/resolve") return { data: resolve };
    if (url.startsWith("/api/projects/p-1/statuses") || url === "/api/statuses") {
      return { data: [{ id: "st-backlog", name: "Backlog", kind: "backlog" }] };
    }
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: plays };
    if (url === "/api/pairing/presence") return { data: { computers: {} } };
    if (url === "/api/pairing/computers") return { data: { computers: [] } };
    if (url === "/api/plays/runs") return { data: runs };
    return { data: [] };
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PlaysRailSection ticket={ticket} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.resetAllMocks();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlaysRailSection", () => {
  it("lists the plays that apply to the ticket's stage, and a play its own rule lets the member run without plays:run in the workspace", async () => {
    mockApi([play], []);
    renderSection();
    expect(await screen.findByRole("heading", { name: "Plays" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /Fix with AI/ })).toBeInTheDocument();
  });

  it("ends each play's row with its own newest run's state, and a play never run with nothing", async () => {
    const other = { ...play, id: "play-2", label: "Review with AI" };
    const run = (id: string, play_id: string, state: string) => ({ id, play_id, state, started_at: new Date().toISOString(), activity: [] });
    mockApi([play, other], ["plays:run"], {}, [run("tr-2", "play-1", "failed"), run("tr-1", "play-1", "done")]);
    renderSection();
    const fix = (await screen.findByRole("button", { name: /Fix with AI/ })).closest("div")!;
    expect(await within(fix).findByRole("img", { name: "failed" })).toBeInTheDocument();
    expect(within(fix).queryByRole("img", { name: "done" })).not.toBeInTheDocument();
    const review = screen.getByRole("button", { name: /Review with AI/ }).closest("div")!;
    expect(within(review).queryByRole("img")).not.toBeInTheDocument();
  });

  it("renders nothing when no play applies", async () => {
    mockApi([]);
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything()));
    expect(screen.queryByRole("heading", { name: "Plays" })).not.toBeInTheDocument();
    expect(screen.queryByText("No plays for this stage.")).not.toBeInTheDocument();
  });

  it("stays hidden for a play the member cannot run", async () => {
    mockApi([{ ...play, can_run: false }]);
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything()));
    expect(screen.queryByRole("heading", { name: "Plays" })).not.toBeInTheDocument();
  });

  it("says once why no play can run, however many plays apply", async () => {
    const second = { ...play, id: "play-2", label: "Review with AI" };
    mockApi([play, second], ["plays:run"], { ok: false, reason: "unpaired" });
    renderSection();
    const hint = "No computer is paired to run plays.";
    expect(await screen.findAllByText(hint)).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Add a computer" })).toHaveAttribute("href", "/settings/pairing");
    expect(screen.getByRole("button", { name: /Fix with AI/ })).toHaveAccessibleDescription(hint);
    expect(screen.getByRole("button", { name: /Review with AI/ })).toBeDisabled();
  });
});
