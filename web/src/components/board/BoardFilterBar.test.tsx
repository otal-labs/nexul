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
  developers: [],
  waitingForMeToTest: false,
  ...overrides,
});

const baseProps = {
  projectId: "p-1",
  developers: [] as string[],
  showWaitingForMeToTest: false,
  filters: filters(),
  onToggleProject: () => {},
  onToggleCategory: () => {},
  onToggleLabel: () => {},
  onSelectType: () => {},
  onToggleStatus: () => {},
  onToggleDeveloper: () => {},
  onToggleWaitingForMeToTest: () => {},
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

  it("renders label, type, and status chips, leaving developers to the avatar stack", async () => {
    const user = userEvent.setup();
    mockReferenceData({ labels: ["bug"] });
    renderFilterBar({
      developers: ["alice"],
      filters: filters({ labels: ["bug"], typeId: "ticket-type-task", statusIds: ["open"], developers: ["alice"] }),
    });
    await openFilterPopover(user);
    expect(await screen.findByRole("button", { name: "bug" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "task" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open" })).toBeInTheDocument();
    expect(screen.queryByText("Developer")).not.toBeInTheDocument();
  });

  it("shows no projects state", async () => {
    mockReferenceData({ projects: [] });
    renderFilterBar();
    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
  });

  it("shows the developer stack beside the filter trigger, marking the selected ones", async () => {
    const user = userEvent.setup();
    const onToggleDeveloper = vi.fn();
    renderFilterBar({ developers: ["alice", "bob"], filters: filters({ developers: ["alice"] }), onToggleDeveloper });
    expect(await screen.findByRole("button", { name: "Developer alice" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Developer bob" })).toHaveAttribute("aria-pressed", "false");
    await user.click(screen.getByRole("button", { name: "Developer bob" }));
    expect(onToggleDeveloper).toHaveBeenCalledWith("bob");
  });

  it("collapses developers past the fifth into a count that opens the rest", async () => {
    const user = userEvent.setup();
    const onToggleDeveloper = vi.fn();
    renderFilterBar({ developers: ["a", "b", "c", "d", "e", "f", "g"], onToggleDeveloper });
    const more = await screen.findByRole("button", { name: "2 more developers" });
    expect(more).toHaveTextContent("+2");
    expect(screen.queryByRole("button", { name: "Developer f" })).not.toBeInTheDocument();
    await user.click(more);
    await user.click(await screen.findByRole("button", { name: "Developer f" }));
    expect(onToggleDeveloper).toHaveBeenCalledWith("f");
  });

  it("moves a selected developer into the visible stack even when they'd otherwise overflow", async () => {
    renderFilterBar({ developers: ["a", "b", "c", "d", "e", "f", "g"], filters: filters({ developers: ["g"] }) });
    expect(await screen.findByRole("button", { name: "Developer g" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByRole("button", { name: "Developer e" })).not.toBeInTheDocument();
  });

  it("offers the waiting-for-me-to-test toggle only when asked, and reports its state", async () => {
    const user = userEvent.setup();
    const onToggleWaitingForMeToTest = vi.fn();
    const { unmount } = renderFilterBar();
    await screen.findByRole("button", { name: /^Filter/ });
    expect(screen.queryByRole("button", { name: /Waiting for me to test/ })).not.toBeInTheDocument();
    unmount();

    renderFilterBar({
      showWaitingForMeToTest: true,
      filters: filters({ waitingForMeToTest: true }),
      onToggleWaitingForMeToTest,
    });
    const toggle = await screen.findByRole("button", { name: /Waiting for me to test/ });
    expect(toggle).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Filter (1)" })).toBeInTheDocument();
    await user.click(toggle);
    expect(onToggleWaitingForMeToTest).toHaveBeenCalled();
  });

  it("renders no stack when no ticket has a developer", async () => {
    renderFilterBar({ developers: [] });
    await screen.findByRole("button", { name: /^Filter/ });
    expect(screen.queryByRole("group", { name: "Filter by developer" })).not.toBeInTheDocument();
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
    const onToggleDeveloper = vi.fn();
    renderFilterBar({
      developers: ["alice"],
      onToggleCategory,
      onToggleLabel,
      onSelectType,
      onToggleStatus,
      onToggleDeveloper,
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
    await user.click(screen.getByRole("button", { name: "Developer alice" }));
    expect(onToggleDeveloper).toHaveBeenCalledWith("alice");
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
