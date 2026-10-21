import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, Route, Routes } from "react-router";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { EmbeddedCrumbsContext } from "@/hooks/useEmbeddedCrumbs";
import { TicketPage } from "@/pages/TicketPage";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { withOwner } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderPage = (entry = "/tickets/t-1", embeddedTicketId?: string, inInbox = false) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <EmbeddedCrumbsContext.Provider value={inInbox ? [] : undefined}>
        <MemoryRouter initialEntries={[entry]}>
          <Routes>
            <Route path="/tickets/:ticketId/:tab?" element={<TicketPage {...(embeddedTicketId ? { ticketId: embeddedTicketId } : {})} />} />
            <Route path="/acme/board/:token" element={<p>Board page</p>} />
          </Routes>
        </MemoryRouter>
      </EmbeddedCrumbsContext.Provider>
    </QueryClientProvider>,
  );
};

const ticketData = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "Add the runner.",
  status: "st-progress",
  number: 7,
  doc_id: "doc-1",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const projectData = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const statusData = [
  { id: "st-backlog", name: "Backlog", kind: "backlog", icon: "", position: 0 },
  { id: "st-progress", name: "In progress", kind: "progress", icon: "", position: 0 },
  { id: "st-done", name: "Shipped", kind: "done", icon: "CircleCheckBig", position: 0 },
];

const reviewData = [
  {
    id: "r-1",
    pr_number: 42,
    repo: "acme/app",
    ticket_id: "t-1",
    doc_id: "",
    status: "approved",
    reviewer: "alice",
    created_at: "2026-08-02T12:00:00Z",
  },
];

// Without an override the viewer is an Owner; an override for the permission read puts a member in their place.
const mockTicket = (overrides: Record<string, unknown> = {}) => {
  const read = withOwner((url: string) => {
    if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
    if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
    if (url === "/api/statuses") return Promise.resolve({ data: statusData });
    if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
    if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
    if (url === "/api/chat/tickets/t-1/notes") return Promise.resolve({ data: { conversation_id: "", notes: [] } });
    return Promise.resolve({ data: [] });
  });
  return vi.mocked(api.get).mockImplementation((url: string) => (url in overrides ? Promise.resolve({ data: overrides[url] }) : read(url)));
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("TicketPage", () => {
  it("renders the ticket with its PREFIX-number id and moves its status from the status control", async () => {
    const user = userEvent.setup();
    mockTicket();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticketData, status: "st-done" } });
    renderPage();
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.getByText("BE-7")).toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: "Status: In progress" }));
    await user.click(await screen.findByRole("button", { name: "Shipped" }));
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "st-done" });
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("shows linked pull requests and their review status", async () => {
    vi.mocked(api.get).mockImplementation(withOwner((url: string) => {
      if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
      if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
      if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
      if (url === "/api/reviews") return Promise.resolve({ data: reviewData });
      if (url === "/api/chat/tickets/t-1/notes") return Promise.resolve({ data: { conversation_id: "", notes: [] } });
      if (url === "/api/tickets/labels") return Promise.resolve({ data: [] });
      if (url === "/api/ticket-types") return Promise.resolve({ data: [] });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
      return Promise.resolve({ data: [] });
    }));
    renderPage();
    expect(await screen.findByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("approved")).toBeInTheDocument();
    expect(screen.getByText("by alice")).toBeInTheDocument();
  });

  it("resolves a /tickets/PREFIX-NUMBER url to the same ticket as its uuid url", async () => {
    vi.mocked(api.get).mockImplementation(withOwner((url: string) => {
      if (url === "/api/tickets") return Promise.resolve({ data: [ticketData] });
      if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
      if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
      if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
      return Promise.resolve({ data: [] });
    }));
    renderPage("/tickets/BE-7");
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.getByText("BE-7")).toBeInTheDocument();
  });

  it("shows not-found for a key that matches no ticket", async () => {
    mockTicket();
    renderPage("/tickets/BE-999");
    expect(await screen.findByRole("heading", { name: "Page not found" })).toBeInTheDocument();
  });

  it("shows the shared error display", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });
});

describe("TicketPage sections", () => {
  it("drops the Thread and the Trail without a workspace", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });
    mockTicket();
    renderPage();

    expect(await screen.findByText("No bugs reported.")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Thread" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Trail" })).not.toBeInTheDocument();
  });

  it("opens on the Ticket tab, its description beside a rail whose every section says something when empty", async () => {
    mockTicket();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Ticket" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Add the runner.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start thread" })).not.toBeInTheDocument();
    expect(await screen.findByText("Pass and Fail show in a testing column.")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(await screen.findByText("No bugs reported.")).toBeInTheDocument();
    expect(await screen.findByText("No play runs yet.")).toBeInTheDocument();
  });

  it("shows the thread on its own tab, which a link opens straight to, in place of the description and rail", async () => {
    const user = userEvent.setup();
    mockTicket();
    const page = renderPage();
    await user.click(await screen.findByRole("tab", { name: "Thread" }));
    expect(await screen.findByRole("button", { name: "Start thread" })).toBeInTheDocument();
    expect(screen.queryByText("Add the runner.")).not.toBeInTheDocument();
    expect(screen.queryByText("No bugs reported.")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    page.unmount();

    renderPage("/tickets/t-1/thread");
    expect(await screen.findByRole("tab", { name: "Thread" })).toHaveAttribute("aria-selected", "true");
    expect(await screen.findByRole("button", { name: "Start thread" })).toBeInTheDocument();
  });

  it("marks the Thread tab while the ticket's run waits on an answer, and clears it once the run goes on", async () => {
    usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
    mockTicket({ "/api/plays/runs/active": { active: { "t-1": "tr-1" }, waiting: { "t-1": "tr-1" } } });
    renderPage();

    expect(await screen.findByRole("tab", { name: "Thread, needs your answer" })).toBeInTheDocument();

    act(() =>
      usePlayRunStore.getState().applyFrame({
        trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", state: "running", activity: null, ended_at: null, last_error: "",
      }),
    );
    expect(screen.getByRole("tab", { name: "Thread" })).toBeInTheDocument();
  });

  it("splits a ticket opened inside another page into the same Ticket and Thread tabs", async () => {
    const user = userEvent.setup();
    mockTicket();
    renderPage("/tickets/t-1", "t-1");
    expect(await screen.findByRole("tab", { name: "Ticket" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Add the runner.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start thread" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Thread" }));
    expect(await screen.findByRole("button", { name: "Start thread" })).toBeInTheDocument();
    expect(screen.queryByText("Add the runner.")).not.toBeInTheDocument();
  });

  it("adds a Notes tab once the ticket has a note, opening the newest with every note in its rail", async () => {
    const user = userEvent.setup();
    const note = (id: string, body: string, file: string, at: string) => ({
      id, conversation_id: "c-1", author_id: "u-1", author_kind: "agent", body, mentions: null, attachment_id: file, created_at: at, updated_at: at,
    });
    mockTicket({
      "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["tickets:read"] },
      "/api/chat/tickets/t-1/notes": {
        conversation_id: "c-1",
        notes: [note("m-2", "Merged in PR #30", "f-2", "2026-10-20T14:11:00Z"), note("m-1", "Left the rollout plan", "f-1", "2026-10-19T09:00:00Z")],
      },
      "/api/attachments": [{ id: "f-2", name: "handoff.md", size: 900, content_type: "text/markdown" }, { id: "f-1", name: "rollout.md", size: 400, content_type: "text/markdown" }],
      "/api/attachments/f-2": "The slot now stores a time zone.",
      "/api/attachments/f-1": "Roll out region by region.",
    });
    renderPage();

    await user.click(await screen.findByRole("tab", { name: "Notes" }));
    expect(await screen.findByRole("heading", { name: "Merged in PR #30" })).toBeInTheDocument();
    expect(await screen.findByText("The slot now stores a time zone.")).toBeInTheDocument();
    expect(screen.queryByText("Add the runner.")).not.toBeInTheDocument();

    const rail = screen.getByRole("region", { name: "Notes" });
    expect(within(rail).getByRole("button", { name: /handoff\.md/ })).toHaveAttribute("aria-current", "true");
    await user.click(within(rail).getByRole("button", { name: /rollout\.md/ }));
    expect(await screen.findByText("Roll out region by region.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Left the rollout plan" })).toBeInTheDocument();
  });

  it("has no Notes tab while the ticket has no note", async () => {
    mockTicket();
    renderPage();
    expect(await screen.findByRole("tab", { name: "Thread" })).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Notes" })).not.toBeInTheDocument();
  });

  it("puts Plays directly under Properties, above Development", async () => {
    mockTicket({
      "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["plays:run"] },
      "/api/workspaces/ws-1/plays/applicable": [{ id: "play-1", label: "Fix with AI", type: "ticket", description: "", show_when_stage: "progress", can_run: true }],
    });
    renderPage();

    const headings = async () => (await screen.findAllByRole("heading", { level: 2 })).map((h) => h.textContent);
    await screen.findByRole("button", { name: /Fix with AI/ });
    const rail = await headings();
    const properties = rail.indexOf("Properties");
    expect(rail.slice(properties, properties + 3)).toEqual(["Properties", "Plays", "Development"]);
  });
});

describe("TicketPage delete", () => {
  const canDelete = { "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["tickets:delete"] } };

  beforeEach(() => {
    vi.mocked(api.delete).mockReset().mockResolvedValue({ data: null });
  });

  it("is not offered to someone without tickets:delete", async () => {
    mockTicket({ "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["tickets:write"] } });
    renderPage();

    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Actions for BE-7" })).not.toBeInTheDocument();
  });

  it("is locked with the reason for someone whose role shows locked actions, and deletes nothing", async () => {
    const user = userEvent.setup();
    mockTicket({ "/api/workspaces/ws-1/me": { role_name: "Member", permissions: ["tickets:write"], show_locked_actions: true } });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Actions for BE-7" }));
    const remove = await screen.findByRole("button", { name: "Delete" });
    expect(remove).toHaveAttribute("aria-disabled", "true");
    await user.click(remove);
    expect(api.delete).not.toHaveBeenCalled();
  });

  it("names the ticket and what goes with it, and deletes nothing when cancelled", async () => {
    const user = userEvent.setup();
    mockTicket(canDelete);
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Actions for BE-7" }));
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Delete BE-7?");
    expect(dialog).toHaveTextContent("Write migrations");
    expect(dialog).toHaveTextContent("labels, links, attachments and thread");
    expect(dialog).toHaveTextContent("can't be undone");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    expect(api.delete).not.toHaveBeenCalled();
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
  });

  it("deletes the ticket and returns to its project's board", async () => {
    const user = userEvent.setup();
    mockTicket(canDelete);
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Actions for BE-7" }));
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Delete ticket" }));

    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1");
    expect(await screen.findByText("Board page")).toBeInTheDocument();
  });

  it("keeps a ticket shown inside another page where it is", async () => {
    const user = userEvent.setup();
    mockTicket(canDelete);
    renderPage("/tickets/t-1", "t-1", true);

    await user.click(await screen.findByRole("button", { name: "Actions for BE-7" }));
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Delete ticket" }));

    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Ticket deleted"));
    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1");
    expect(screen.queryByText("Board page")).not.toBeInTheDocument();
  });
});

describe("TicketPage attachments", () => {
  // Writes tickets in p-1 and only reads in p-2; the sidebar's project is where the viewer happens to be looking.
  const me = {
    role_name: "Member",
    permissions: ["tickets:read"],
    show_locked_actions: true,
    projects: [
      { project_id: "p-1", actions: ["tickets:write"] },
      { project_id: "p-2", actions: [] },
    ],
  };

  it.each([
    { name: "the ticket's own project lets them write, though another is selected", selected: "p-2", ticketProject: "p-1", offered: true, inInbox: false },
    { name: "the selected project lets them write, but the ticket is in another", selected: "p-1", ticketProject: "p-2", offered: false, inInbox: false },
    { name: "a ticket opened from the Inbox follows its own project", selected: "p-2", ticketProject: "p-1", offered: true, inInbox: true },
  ])("$name", async ({ selected, ticketProject, offered, inInbox }) => {
    useWorkspaceStore.setState({ selectedProjectId: selected });
    mockTicket({ "/api/workspaces/ws-1/me": me, "/api/tickets/t-1": { ...ticketData, project_id: ticketProject } });
    renderPage("/tickets/t-1", inInbox ? "t-1" : undefined, inInbox);

    const add = await screen.findByRole("button", { name: "Add attachment" });
    if (offered) expect(add).not.toHaveAttribute("aria-disabled");
    if (!offered) expect(add).toHaveAttribute("aria-disabled", "true");
  });
});
