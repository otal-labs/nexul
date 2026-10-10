import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketThreadSection } from "@/components/chat/TicketThreadSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { playQueue, queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversation = {
  id: "c1",
  workspace_id: "ws-1",
  kind: "ticket_thread",
  ticket_id: "t1",
  created_by: "u1",
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<TicketThreadSection workspaceId="ws-1" ticketId="t1" />, { wrapper });
};

const mockGetByUrl = (routes: Record<string, unknown>) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    for (const [prefix, data] of Object.entries(routes)) {
      if (url.startsWith(prefix)) return { data };
    }
    return { data: [] };
  });
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("TicketThreadSection", () => {
  it("shows a lazy Start thread button when the ticket has no thread yet", async () => {
    mockGetByUrl({ "/api/chat/tickets/thread-status": { t1: false } });
    renderSection();
    expect(await screen.findByRole("button", { name: /start thread/i })).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });

  it("shows what auto plays skipped or didn't run above Start thread while the ticket has no thread", async () => {
    mockGetByUrl({
      "/api/chat/tickets/thread-status": { t1: false },
      "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["plays:run"] },
      "/api/workspaces/ws-1/plays": [{ id: "play-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket", description: "", instructions: "", enabled: true, show_when_stage: "progress", excluded_project_ids: [] }],
      "/api/plays/queue": playQueue({
        items: [
          queueItem({ target_id: "t1", person_id: "", status: "didnt_run", reason: "nobody to run it on: the ticket has no developer", decided_at: "2026-10-10T09:00:00Z" }),
          queueItem({ id: "q-2", target_id: "t1", play_id: "play-2", play_label: "Review", status: "skipped", reason: "no longer unblocked", decided_at: "2026-10-10T08:00:00Z" }),
        ],
      }),
    });
    renderSection();

    expect(await screen.findByText(/nobody is the ticket's developer/)).toHaveTextContent("Fix with AI didn't run: nobody is the ticket's developer");
    expect(screen.getByText(/no longer unblocked/)).toHaveTextContent("Review skipped: no longer unblocked");
    expect(await screen.findByRole("button", { name: "Run Fix with AI" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start thread/i })).toBeInTheDocument();
  });

  it("creates the thread on Start thread and renders the conversation", async () => {
    const user = userEvent.setup();
    mockGetByUrl({
      "/api/chat/tickets/thread-status": { t1: false },
      "/api/auth/me": meResponse,
      "/api/chat/conversations/c1/messages": [],
      "/api/workspaces": { members: [], invites: [] },
    });
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/chat/tickets/t1/thread") return { data: conversation };
      return { data: {} };
    });
    renderSection();

    await user.click(await screen.findByRole("button", { name: /start thread/i }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/chat/tickets/t1/thread", { workspace_id: "ws-1" }));
    expect(await screen.findByLabelText("Message")).toBeInTheDocument();
  });

  it("auto-loads an existing thread without requiring a click", async () => {
    mockGetByUrl({
      "/api/chat/tickets/thread-status": { t1: true },
      "/api/auth/me": meResponse,
      "/api/chat/conversations/c1/messages": [],
      "/api/workspaces": { members: [], invites: [] },
    });
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/chat/tickets/t1/thread") return { data: conversation };
      return { data: {} };
    });
    renderSection();

    expect(screen.queryByRole("button", { name: /start thread/i })).not.toBeInTheDocument();
    expect(await screen.findByLabelText("Message")).toBeInTheDocument();
  });
});
