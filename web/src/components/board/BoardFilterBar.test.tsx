import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardFilterBar, type BoardFilters } from "@/components/board/BoardFilterBar";
import type { StatusKind } from "@/models/Status";

// BoardFilterBar fetches its own reference data via TanStack Query hooks, so
// tests need a QueryClient ancestor, a mocked api client, and a route with a
// :projectId param (ticket types/statuses are project-scoped).
vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const categories = [
  { id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, created_at: "", updated_at: "" },
];

const statuses = [
  { id: "open", name: "Open", position: 0, kind: "progress" as StatusKind, icon: "", created_at: "", updated_at: "" },
];

const ticketTypes = [
  { id: "ticket-type-task", name: "task", position: 0, color: "", created_at: "", updated_at: "" },
];

const mockReferenceData = ({
  projects: projectList = projects,
  categories: categoryList = categories,
  labels = [] as string[],
  ticketTypes: typeList = ticketTypes,
  statuses: statusList = statuses,
} = {}) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: projectList };
    if (url === "/api/categories") return { data: categoryList };
    if (url === "/api/tickets/labels") return { data: labels };
    if (url === "/api/ticket-types") return { data: typeList };
    if (url === "/api/statuses") return { data: statusList };
    return { data: [] };
  });
};

const filters = (overrides: Partial<BoardFilters> = {}): BoardFilters => ({
  projectIds: [],
  categoryId: null,
  labels: [],
  typeId: null,
  statusIds: [],
  assignees: [],
  ...overrides,
});

const baseProps = {
  projectId: "p-1",
  assignees: [] as string[],
  filters: filters(),
  onToggleProject: () => {},
  onToggleCategory: () => {},
  onToggleLabel: () => {},
  onSelectType: () => {},
  onToggleStatus: () => {},
  onToggleAssignee: () => {},
  onClear: () => {},
  onNewTicket: () => {},
  onNewCategory: () => {},
};

const renderFilterBar = (props: Partial<Parameters<typeof BoardFilterBar>[0]> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/board/p-1"]}>
        <Routes>
          <Route path="/board/:projectId" element={<BoardFilterBar {...baseProps} {...props} />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const openFilterPopover = async (user: ReturnType<typeof userEvent.setup>) =>
  user.click(await screen.findByRole("button", { name: /^Filter/ }));

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  mockReferenceData();
});

describe("BoardFilterBar", () => {
  it("says nothing about projects when there's nothing to say — no filter applied is just the default", async () => {
    renderFilterBar();
    await screen.findByRole("button", { name: /^Filter/ });
    expect(screen.queryByText("Showing all projects")).not.toBeInTheDocument();
  });

  it("renders the create menu next to the filter trigger", async () => {
    renderFilterBar();
    expect(await screen.findByRole("button", { name: "Add" })).toBeInTheDocument();
  });

  it("hides the Projects filter row and toggle when the board is project-scoped", async () => {
    const user = userEvent.setup();
    renderFilterBar({ hideProjectFilter: true });
    await openFilterPopover(user);
    expect(screen.queryByText("Projects")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Filter by Backend" })).not.toBeInTheDocument();
  });

  it("toggles a project filter", async () => {
    const user = userEvent.setup();
    const onToggleProject = vi.fn();
    renderFilterBar({ onToggleProject });
    await openFilterPopover(user);
    await user.click(await screen.findByRole("button", { name: "Filter by Backend" }));
    expect(onToggleProject).toHaveBeenCalledWith("p-1");
  });

  it("shows active filters and clears them", async () => {
    const user = userEvent.setup();
    const onClear = vi.fn();
    renderFilterBar({ filters: filters({ projectIds: ["p-1"], categoryId: "c-1" }), onClear });
    expect(await screen.findByRole("button", { name: "Filter (2)" })).toBeInTheDocument();
    await openFilterPopover(user);
    expect(screen.getByRole("button", { name: "Filter by Backend" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Sprint 1" })).toHaveAttribute("aria-pressed", "true");
    // "Filter (2)" on the trigger already says a filter is active — no
    // separate "Filtered" label to duplicate it.
    expect(screen.queryByText("Filtered")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Clear filter/ }));
    expect(onClear).toHaveBeenCalled();
  });

  it("renders label, type, and status chips, leaving assignees to the avatar stack", async () => {
    const user = userEvent.setup();
    mockReferenceData({ labels: ["bug"] });
    renderFilterBar({
      assignees: ["alice"],
      filters: filters({ labels: ["bug"], typeId: "ticket-type-task", statusIds: ["open"], assignees: ["alice"] }),
    });
    await openFilterPopover(user);
    expect(await screen.findByRole("button", { name: "bug" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "task" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open" })).toBeInTheDocument();
    expect(screen.queryByText("Assignee")).not.toBeInTheDocument();
  });

  it("shows no projects state", async () => {
    mockReferenceData({ projects: [] });
    renderFilterBar();
    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
  });

  it("shows the assignee stack beside the filter trigger, marking the selected ones", async () => {
    const user = userEvent.setup();
    const onToggleAssignee = vi.fn();
    renderFilterBar({ assignees: ["alice", "bob"], filters: filters({ assignees: ["alice"] }), onToggleAssignee });
    expect(await screen.findByRole("button", { name: "Assignee alice" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Assignee bob" })).toHaveAttribute("aria-pressed", "false");
    await user.click(screen.getByRole("button", { name: "Assignee bob" }));
    expect(onToggleAssignee).toHaveBeenCalledWith("bob");
  });

  it("collapses assignees past the fifth into a count that opens the rest", async () => {
    const user = userEvent.setup();
    const onToggleAssignee = vi.fn();
    renderFilterBar({ assignees: ["a", "b", "c", "d", "e", "f", "g"], onToggleAssignee });
    const more = await screen.findByRole("button", { name: "2 more assignees" });
    expect(more).toHaveTextContent("+2");
    expect(screen.queryByRole("button", { name: "Assignee f" })).not.toBeInTheDocument();
    await user.click(more);
    await user.click(await screen.findByRole("button", { name: "Assignee f" }));
    expect(onToggleAssignee).toHaveBeenCalledWith("f");
  });

  it("moves a selected assignee into the visible stack even when they'd otherwise overflow", async () => {
    renderFilterBar({ assignees: ["a", "b", "c", "d", "e", "f", "g"], filters: filters({ assignees: ["g"] }) });
    expect(await screen.findByRole("button", { name: "Assignee g" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByRole("button", { name: "Assignee e" })).not.toBeInTheDocument();
  });

  it("renders no stack when no ticket has an assignee", async () => {
    renderFilterBar({ assignees: [] });
    await screen.findByRole("button", { name: /^Filter/ });
    expect(screen.queryByRole("group", { name: "Filter by assignee" })).not.toBeInTheDocument();
  });
});

describe("BoardFilterBar interactions", () => {
  it("toggles category, label, type, and status chips", async () => {
    const user = userEvent.setup();
    mockReferenceData({ labels: ["bug"] });
    const onToggleCategory = vi.fn();
    const onToggleLabel = vi.fn();
    const onSelectType = vi.fn();
    const onToggleStatus = vi.fn();
    const onToggleAssignee = vi.fn();
    renderFilterBar({
      assignees: ["alice"],
      onToggleCategory,
      onToggleLabel,
      onSelectType,
      onToggleStatus,
      onToggleAssignee,
    });
    await openFilterPopover(user);
    await user.click(await screen.findByRole("button", { name: "Sprint 1" }));
    expect(onToggleCategory).toHaveBeenCalledWith("c-1");
    await user.click(screen.getByRole("button", { name: "bug" }));
    expect(onToggleLabel).toHaveBeenCalledWith("bug");
    await user.click(screen.getByRole("button", { name: "task" }));
    expect(onSelectType).toHaveBeenCalledWith("ticket-type-task");
    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(onToggleStatus).toHaveBeenCalledWith("open");
    await user.click(screen.getByRole("button", { name: "Assignee alice" }));
    expect(onToggleAssignee).toHaveBeenCalledWith("alice");
  });

  it("toggles the Uncategorized chip", async () => {
    const user = userEvent.setup();
    const onToggleCategory = vi.fn();
    renderFilterBar({ onToggleCategory });
    await openFilterPopover(user);
    await user.click(await screen.findByRole("button", { name: "Uncategorized" }));
    expect(onToggleCategory).toHaveBeenCalledWith("uncategorized");
  });
});

describe("BoardFilterBar all/clear toggles", () => {
  it("toggles the active category chip off", async () => {
    const user = userEvent.setup();
    const onToggleCategory = vi.fn();
    renderFilterBar({ filters: filters({ categoryId: "c-1" }), onToggleCategory });
    await openFilterPopover(user);
    await user.click(await screen.findByRole("button", { name: "Sprint 1" }));
    expect(onToggleCategory).toHaveBeenCalledWith(null);
  });
});
