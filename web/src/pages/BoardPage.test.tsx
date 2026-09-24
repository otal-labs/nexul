import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardPage } from "@/pages/BoardPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { pickOption } from "@/test/pickOption";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const categories = [
  { id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, created_at: "", updated_at: "" },
];

const statuses = [
  { id: "open", name: "Open", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" },
  { id: "done", name: "Done", position: 1, kind: "done", icon: "", created_at: "", updated_at: "" },
];

const ticketTypes = [
  { id: "ticket-type-task", name: "task", position: 0, created_at: "", updated_at: "" },
];

const ticket = (
  id: string,
  projectId: string,
  title: string,
  status: string,
  categoryId = "",
  developer = "",
) => ({
  id,
  project_id: projectId,
  category_id: categoryId,
  type_id: "ticket-type-task",
  title,
  body: "",
  status,
  position: 0,
  number: 1,
  doc_id: "",
  developer,
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
});

const openFilterPopover = async (user: ReturnType<typeof userEvent.setup>) =>
  user.click(await screen.findByRole("button", { name: /^Filter/ }));

const openCreateMenuItem = async (user: ReturnType<typeof userEvent.setup>, label: "New ticket" | "New category") => {
  await user.click(await screen.findByRole("button", { name: "Add" }));
  await user.click(await screen.findByRole("button", { name: label }));
};

// Every render of the board needs a resolved projectId (ticket 08), so most behavioral tests render the scoped route directly; the unscoped "/board" route is exercised in "BoardPage routing" below.
const renderPage = (initialPath = "/board/p-1") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/board" element={<BoardPage />} />
          <Route path="/board/:projectId" element={<BoardPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const mockGet = (tickets: unknown[], projectList: typeof projects = projects, categoryList: typeof categories = categories) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: projectList };
    if (url === "/api/auth/me") return { data: { user: { login: "onik97" } } };
    const projectMatch = /^\/api\/projects\/([^/]+)$/.exec(url);
    if (projectMatch) return { data: projectList.find((p) => p.id === projectMatch[1]) };
    if (url.startsWith("/api/categories")) return { data: categoryList };
    if (url.startsWith("/api/statuses")) return { data: statuses };
    if (url.startsWith("/api/ticket-types")) return { data: ticketTypes };
    if (url.startsWith("/api/tickets/labels")) return { data: [] };
    return { data: tickets };
  });
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  // Last-viewed project is read from this store (ticket 08); start every test from a clean slate so redirect targets are deterministic.
  useWorkspaceStore.setState({ selectedWorkspaceId: "", selectedProjectId: "" });
  useWorkspaceStore.persist.clearStorage();
  localStorage.clear();
});

describe("BoardPage", () => {
  it("renders swimlanes grouped by category, scoped to the URL's project", async () => {
    mockGet([
      ticket("t-1", "p-1", "Fix login", "open", "c-1"),
      ticket("t-2", "p-2", "Wire FTS", "done"),
    ]);
    renderPage();
    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Sprint 1 swimlane" })).toBeInTheDocument();
    expect(screen.queryByText("Wire FTS")).not.toBeInTheDocument();
  });

  it("resolves a prefix URL (/board/BE) to the same project as its id, case-insensitively", async () => {
    mockGet([ticket("t-1", "p-1", "Fix login", "open")]);
    renderPage("/board/be");
    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Backend")).toBeInTheDocument();
  });

  it("shows a not-found state for a token matching no project, instead of an empty board", async () => {
    mockGet([]);
    renderPage("/board/project-general-randomstring");
    expect(await screen.findByText("Project not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /go to your board/i })).toHaveAttribute("href", "/board");
  });

  it("filters by category", async () => {
    const user = userEvent.setup();
    mockGet([
      ticket("t-1", "p-1", "Fix login", "open", "c-1"),
      ticket("t-2", "p-1", "Wire FTS", "open"),
    ]);
    renderPage();

    await openFilterPopover(user);
    await user.click(screen.getByRole("button", { name: "Uncategorized" }));
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
    expect(screen.queryByText("Fix login")).not.toBeInTheDocument();
  });

  // Pointer/touch/keyboard drags aren't reproducible through jsdom; the drag-end mapping is covered in dragMove.test.ts and the resulting API calls in TicketHooks.test.tsx/CategoryHooks.test.tsx. This just proves BoardPage renders the drop targets.
  it("renders a status column and a swimlane as drop targets for a ticket", async () => {
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1")]);
    renderPage();

    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Done column in Sprint 1" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Sprint 1 swimlane" })).toBeInTheDocument();
  });

  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("Failed to load the board.")).toBeInTheDocument();
  });

  it("creates a ticket through the relocated dialog", async () => {
    const user = userEvent.setup();
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1")]);
    vi.mocked(api.post).mockResolvedValue({
      data: ticket("t-2", "p-1", "Ship board create", "open"),
    });
    renderPage();

    await openCreateMenuItem(user, "New ticket");
    await user.type(await screen.findByLabelText("Title"), "Ship board create");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets", {
      title: "Ship board create",
      body: "",
      project_id: "p-1",
      doc_id: "",
      developer: "",
      tester: "",
      category_id: "",
      type_id: "ticket-type-task",
    });
  });

  it("creates a category for the scoped project and shows it as a swimlane", async () => {
    const user = userEvent.setup();
    // Reassigned (not pushed) on create so the refetch returns a fresh array — React Query's structural sharing would swallow an in-place mutation of the same reference.
    let currentCategories = [...categories];
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { login: "onik97" } } };
      if (url.startsWith("/api/projects")) return { data: projects };
      if (url.startsWith("/api/categories")) return { data: currentCategories };
      if (url.startsWith("/api/statuses")) return { data: statuses };
      if (url.startsWith("/api/ticket-types")) return { data: ticketTypes };
      if (url.startsWith("/api/tickets/labels")) return { data: [] };
      return { data: [ticket("t-1", "p-1", "Fix login", "open", "c-1")] };
    });
    vi.mocked(api.post).mockImplementation((url: string, body) => {
      if (url === "/api/categories") {
        const created = {
          id: "c-2",
          ...(body as { project_id: string; name: string }),
          position: 1,
          created_at: "",
          updated_at: "",
        };
        currentCategories = [...currentCategories, created];
        return Promise.resolve({ data: created });
      }
      return Promise.resolve({ data: {} });
    });
    renderPage();

    await openCreateMenuItem(user, "New category");
    await pickOption(user, "Project", "Backend");
    await user.type(screen.getByLabelText("Name"), "Sprint 2");
    await user.click(screen.getByRole("button", { name: "Create category" }));

    expect(api.post).toHaveBeenCalledWith("/api/categories", { project_id: "p-1", name: "Sprint 2", color: "" });
    expect(await screen.findByRole("region", { name: "Sprint 2 swimlane" })).toBeInTheDocument();
  });

  it("does not show a swimlane for a category outside the scoped project", async () => {
    const user = userEvent.setup();
    let currentCategories = [...categories];
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { login: "onik97" } } };
      if (url.startsWith("/api/projects")) return { data: projects };
      if (url.startsWith("/api/categories")) return { data: currentCategories };
      if (url.startsWith("/api/statuses")) return { data: statuses };
      if (url.startsWith("/api/ticket-types")) return { data: ticketTypes };
      if (url.startsWith("/api/tickets/labels")) return { data: [] };
      return { data: [ticket("t-1", "p-1", "Fix login", "open", "c-1")] };
    });
    vi.mocked(api.post).mockImplementation((url: string, body) => {
      if (url === "/api/categories") {
        const created = {
          id: "c-2",
          ...(body as { project_id: string; name: string }),
          position: 1,
          created_at: "",
          updated_at: "",
        };
        currentCategories = [...currentCategories, created];
        return Promise.resolve({ data: created });
      }
      return Promise.resolve({ data: {} });
    });
    renderPage("/board/p-1");
    expect(await screen.findByText("Fix login")).toBeInTheDocument();

    await openCreateMenuItem(user, "New category");
    await pickOption(user, "Project", "Frontend");
    await user.type(screen.getByLabelText("Name"), "Sprint 2");
    await user.click(screen.getByRole("button", { name: "Create category" }));

    expect(api.post).toHaveBeenCalledWith("/api/categories", { project_id: "p-2", name: "Sprint 2", color: "" });
    expect(screen.queryByRole("region", { name: "Sprint 2 swimlane" })).not.toBeInTheDocument();
  });

  it("shows the create-first-ticket empty state on a pristine board — the URL scope is not a filter", async () => {
    mockGet([], projects, []);
    renderPage();
    expect(await screen.findByText("No tickets yet — create the first one.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Filter" })).toBeInTheDocument();
  });

  it("shows a ticketless category as an empty swimlane on a fresh load", async () => {
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1")], projects, [
      ...categories,
      { id: "c-2", project_id: "p-1", name: "Sprint 2", position: 1, created_at: "", updated_at: "" },
    ]);
    renderPage();
    expect(await screen.findByRole("region", { name: "Sprint 2 swimlane" })).toBeInTheDocument();
  });

  it("does not show the empty state while tickets are still loading", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { login: "onik97" } } };
      if (url.startsWith("/api/projects")) return { data: projects };
      if (url.startsWith("/api/categories")) return { data: categories };
      if (url.startsWith("/api/statuses")) return { data: statuses };
      if (url.startsWith("/api/ticket-types")) return { data: ticketTypes };
      if (url.startsWith("/api/tickets/labels")) return { data: [] };
      return new Promise(() => {});
    });
    renderPage();
    expect(
      screen.queryByText("No tickets match the active filters."),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Filter" })).not.toBeInTheDocument();
  });
});

describe("BoardPage project-scoped route", () => {
  it("scopes to the project in the URL, hides the Projects filter, and titles the page with it", async () => {
    const user = userEvent.setup();
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1"), ticket("t-2", "p-2", "Wire FTS", "open")]);
    renderPage("/board/p-1");

    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(screen.queryByText("Wire FTS")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Backend" })).toBeInTheDocument();
    await openFilterPopover(user);
    expect(screen.queryByRole("button", { name: "Filter by Backend" })).not.toBeInTheDocument();
  });
});

// The unscoped /board route redirects to a real project's board rather than rendering a mixed board, since statuses/ticket_types are project-scoped (ticket 08).
describe("BoardPage routing", () => {
  it("redirects to the last-viewed project from the workspace store", async () => {
    useWorkspaceStore.getState().selectProject("p-2");
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1"), ticket("t-2", "p-2", "Wire FTS", "open")]);
    renderPage("/board");

    expect(await screen.findByText("Wire FTS")).toBeInTheDocument();
    expect(screen.queryByText("Fix login")).not.toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Frontend" })).toBeInTheDocument();
  });

  it("falls back to the first project by position when nothing is remembered", async () => {
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1"), ticket("t-2", "p-2", "Wire FTS", "open")]);
    renderPage("/board");

    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Backend" })).toBeInTheDocument();
  });

  it("falls back to the first project by position when the remembered project no longer exists", async () => {
    useWorkspaceStore.getState().selectProject("p-deleted");
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1"), ticket("t-2", "p-2", "Wire FTS", "open")]);
    renderPage("/board");

    expect(await screen.findByText("Fix login")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Backend" })).toBeInTheDocument();
  });

  it("shows the create-first-project empty state when the workspace has zero projects", async () => {
    mockGet([], []);
    renderPage("/board");

    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New project" })).toBeInTheDocument();
  });

  it("remembers the project it lands on for the next unscoped visit", async () => {
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1")]);
    renderPage("/board/p-2");

    await screen.findByRole("heading", { name: "Frontend" });
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("shows an error state when the project list itself fails to load", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage("/board");
    expect(await screen.findByText("Failed to load the board.")).toBeInTheDocument();
  });
});

describe("BoardPage filters", () => {
  it("filters by status", async () => {
    const user = userEvent.setup();
    mockGet([ticket("t-1", "p-1", "Fix login", "open", "c-1"), ticket("t-2", "p-1", "Wire FTS", "done", "c-1")]);
    renderPage();
    await openFilterPopover(user);
    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
    expect(screen.queryByText("Fix login")).not.toBeInTheDocument();
  });

  it("filters by developer", async () => {
    const user = userEvent.setup();
    mockGet([
      ticket("t-1", "p-1", "Fix login", "open", "c-1", "alice"),
      ticket("t-2", "p-1", "Wire FTS", "open", "c-1", "bob"),
    ]);
    renderPage();
    // Developers filter from the avatar stack beside the Filter button, not from the popover.
    await user.click(await screen.findByRole("button", { name: "Developer alice" }));
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.queryByText("Wire FTS")).not.toBeInTheDocument();
  });

  it("clears all filters", async () => {
    const user = userEvent.setup();
    mockGet([
      ticket("t-1", "p-1", "Fix login", "open", "c-1"),
      ticket("t-2", "p-1", "Wire FTS", "done", "c-1"),
    ]);
    renderPage();
    await openFilterPopover(user);
    await user.click(screen.getByRole("button", { name: "Done" }));
    await user.click(screen.getByRole("button", { name: /Clear filter/ }));
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });
});

describe("BoardPage label filter", () => {
  it("filters by label", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { login: "onik97" } } };
      if (url.startsWith("/api/projects")) return { data: projects };
      if (url.startsWith("/api/categories")) return { data: categories };
      if (url.startsWith("/api/statuses")) return { data: statuses };
      if (url.startsWith("/api/ticket-types")) return { data: ticketTypes };
      if (url.startsWith("/api/tickets/labels")) return { data: ["bug"] };
      return {
        data: [
          { ...ticket("t-1", "p-1", "Fix login", "open", "c-1"), labels: ["bug"] },
          { ...ticket("t-2", "p-1", "Wire FTS", "open", "c-1"), labels: [] },
        ],
      };
    });
    renderPage();

    await openFilterPopover(user);
    await user.click(screen.getByRole("button", { name: "bug" }));
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.queryByText("Wire FTS")).not.toBeInTheDocument();
  });
});
