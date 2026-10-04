import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { LiveSocket } from "@/api/ws";
import { TicketDetail } from "@/components/ticket/TicketDetail";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";
import { TicketStatus, type Ticket } from "@/models/Ticket";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn().mockResolvedValue({ data: [] }) },
  resolveWSBase: vi.fn(() => "ws://test"),
  errorMessage: vi.fn(() => "error"),
}));

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  send = vi.fn();
  close = vi.fn();
  dispatch(raw: string) {
    this.onmessage?.({ data: raw });
  }
}

const commitsOf = (socket: FakeSocket) =>
  socket.send.mock.calls
    .map((c) => JSON.parse(c[0] as string) as { type: string; title?: string })
    .filter((f) => f.type === "commit");

const writer = ["tickets:read", "tickets:write"];
const reader = ["tickets:read"];

const renderDetail = async (ticket: Ticket, opts: { project?: Project; permissions?: string[] } = {}) => {
  const socket = new FakeSocket();
  const urls: string[] = [];
  // Primed queries never refetch, so the blanket api mock cannot clobber the role or profile.
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: opts.permissions ?? reader });
  const result = render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <TicketDetail
          ticket={ticket}
          {...(opts.project ? { project: opts.project } : {})}
          wsFactory={(url) => {
            urls.push(url);
            return socket;
          }}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  // connect() defers a tick so StrictMode's throwaway session never opens a socket.
  await act(() => new Promise((r) => setTimeout(r, 0)));
  return { ...result, socket, urls };
};

beforeEach(() => {
  useSessionStore.setState({ token: "token-1", isLoggedIn: true });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "Add the migration runner.",
  status: TicketStatus.InProgress,
  position: 0,
  number: 1,
  doc_id: "doc-1",
  developer: "onik97",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const project: Project = {
  id: "p-1",
  name: "Backend",
  prefix: "BE",
  position: 0,
  icon: "",
  tests_location: "",
  created_at: "",
  updated_at: "",
};

describe("TicketDetail", () => {
  it("shows a reader the static title and body and opens no live room", async () => {
    const { urls } = await renderDetail(ticket);
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(await screen.findByText("Add the migration runner.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Ticket title" })).not.toBeInTheDocument();
    expect(urls).toHaveLength(0);
  });

  it("names the reporter in the meta line, as Nexul for a person when an agent filed it", async () => {
    const { unmount } = await renderDetail(ticket);
    expect(screen.getByText(/ by onik97$/)).toBeInTheDocument();
    unmount();
    await renderDetail({ ...ticket, reporter: { kind: "user:mcp", login: "lena" } });
    expect(screen.getByText(/ by Nexul · from lena$/)).toBeInTheDocument();
  });

  it("falls back to the raw ticket id when no project is provided", async () => {
    await renderDetail(ticket);
    expect(screen.getByText("t-1")).toBeInTheDocument();
  });

  it("renders PREFIX-number instead of the raw id when a project is provided", async () => {
    await renderDetail(ticket, { project });
    expect(screen.getByText("BE-1")).toBeInTheDocument();
    expect(screen.queryByText("t-1")).not.toBeInTheDocument();
  });

  it("edits a writer's ticket live in its own room, with no Save button", async () => {
    const { urls, socket } = await renderDetail(ticket, { permissions: writer });
    socket.onopen?.({});
    expect(urls[0]).toContain("/ws/collab/tickets/t-1?mode=edit");
    expect(screen.getByRole("textbox", { name: "Ticket title" })).toHaveValue("Write migrations");
    // The e2e suite finds tickets by heading name, so the <h1> takes its name from the input's value.
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(await screen.findByLabelText("Ticket description")).toBeInTheDocument();
    expect(await screen.findByText("Live")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /save/i })).not.toBeInTheDocument();
  });

  it("Enter commits the rename as one line, carrying the typed title", async () => {
    const user = userEvent.setup();
    const { socket } = await renderDetail(ticket, { permissions: writer });
    socket.onopen?.({});
    await act(async () => {
      socket.dispatch(JSON.stringify({ type: "init", seq: 0, snapshot: null, updates: [], presence: [] }));
    });

    const title = screen.getByRole("textbox", { name: "Ticket title" });
    await user.clear(title);
    await user.click(title);
    await user.paste("First line\nSecond line");
    await user.keyboard("{Enter}");

    expect(title).toHaveValue("First line Second line");
    expect(commitsOf(socket).at(-1)?.title).toBe("First line Second line");
  });
});
