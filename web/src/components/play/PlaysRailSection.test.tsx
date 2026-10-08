import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlaysRailSection } from "@/components/play/PlaysRailSection";
import type { Play } from "@/models/Play";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket = { id: "t-1", project_id: "p-1", status: "st-backlog" } as Ticket;

const play = { id: "play-1", label: "Fix with AI", type: "ticket", description: "Fix it", show_when_stage: "backlog" } as Play;

const mockApi = (plays: Play[], permissions = ["plays:run"]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url.startsWith("/api/projects/p-1/statuses") || url === "/api/statuses") {
      return { data: [{ id: "st-backlog", name: "Backlog", kind: "backlog" }] };
    }
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: plays };
    if (url === "/api/pairing/presence") return { data: { computers: {} } };
    if (url === "/api/pairing/computers") return { data: { computers: [] } };
    return { data: [] };
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PlaysRailSection ticket={ticket} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.resetAllMocks();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlaysRailSection", () => {
  it("lists the plays that apply to the ticket's stage", async () => {
    mockApi([play]);
    renderSection();
    expect(await screen.findByRole("heading", { name: "Plays" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /Fix with AI/ })).toBeInTheDocument();
  });

  it("renders nothing when no play applies", async () => {
    mockApi([]);
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything()));
    expect(screen.queryByRole("heading", { name: "Plays" })).not.toBeInTheDocument();
    expect(screen.queryByText("No plays for this stage.")).not.toBeInTheDocument();
  });

  it("stays hidden for a member who cannot run plays", async () => {
    mockApi([play], []);
    renderSection();
    await waitFor(() => {
      expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything());
      expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me");
    });
    expect(screen.queryByRole("heading", { name: "Plays" })).not.toBeInTheDocument();
  });
});
