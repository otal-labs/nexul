import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { KanbanColumn } from "@/components/board/KanbanColumn";
import type { BoardStatus } from "@/models/Status";
import type { Ticket, TicketStatus } from "@/models/Ticket";

// TicketCard (T10) fetches ticket types and label colors via TanStack Query, so every render
// needs a QueryClient ancestor; empty-array responses suffice since this file doesn't assert on badge colors (see TicketCard.test.tsx).
vi.mock("@/api/client", () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

// F6: TicketCard navigates directly via useNavigate, not an onSelect callback through KanbanColumn.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => mockNavigate };
});

const column: BoardStatus = {
  id: "open",
  name: "Open",
  position: 0,
  kind: "progress",
  icon: "",
  created_at: "",
  updated_at: "",
};

const ticket = (id: string, title: string, status: string, position = 0): Ticket => ({
  id,
  project_id: "p-1",
  category_id: "c-1",
  type_id: "ticket-type-task",
  title,
  body: "",
  status: status as TicketStatus,
  position,
  number: 1,
  doc_id: "",
  developer: "onik97",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
});

const renderColumn = (props: Partial<Parameters<typeof KanbanColumn>[0]> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <KanbanColumn
        droppableId="column-Sprint 1-open"
        laneLabel="Sprint 1"
        categoryId="c-1"
        column={column}
        tickets={[ticket("t-1", "Fix login", "open")]}
        onAddTicket={() => {}}
        {...props}
      />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockNavigate.mockReset();
});

describe("KanbanColumn", () => {
  it("renders the column name and ticket count", () => {
    renderColumn();
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("renders a card per ticket", () => {
    renderColumn({ tickets: [ticket("t-1", "Fix login", "open"), ticket("t-2", "Wire FTS", "open")] });
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });

  it("shows the empty hint when no tickets match", () => {
    renderColumn({ tickets: [] });
    expect(screen.getByText("No tickets")).toBeInTheDocument();
  });

  it("navigates to the ticket detail page on click", async () => {
    const user = userEvent.setup();
    renderColumn();
    await user.click(screen.getByText("Fix login"));
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/t-1");
  });

  it("registers as a drop target scoped to the lane and column", () => {
    renderColumn();
    expect(screen.getByRole("region", { name: "Open column in Sprint 1" })).toBeInTheDocument();
  });

  it("calls onAddTicket with this column's status id when the + button is clicked", async () => {
    const user = userEvent.setup();
    const onAddTicket = vi.fn();
    renderColumn({ onAddTicket });
    await user.click(screen.getByRole("button", { name: "New ticket in Open" }));
    expect(onAddTicket).toHaveBeenCalledWith("open");
  });

  it("exposes a grip that reorders the column, separate from the add button", () => {
    renderColumn();
    const grip = screen.getByRole("button", { name: "Reorder Open" });
    expect(grip).toHaveAttribute("aria-roledescription", "sortable");
    expect(screen.getByRole("button", { name: "New ticket in Open" })).not.toBe(grip);
  });

  it("falls back to the stage dot when no icon is set", () => {
    renderColumn({ column: { ...column, icon: "" } });
    const heading = screen.getByRole("heading", { level: 4 });
    expect(heading.querySelector("span > svg")).not.toBeInTheDocument();
    expect(heading.querySelector(".rounded-full.bg-info")).toBeInTheDocument();
  });

  it("renders the chosen icon instead of the dot when set", () => {
    renderColumn({ column: { ...column, icon: "CircleCheckBig" } });
    const heading = screen.getByRole("heading", { level: 4 });
    expect(heading.querySelector(".rounded-full.bg-info")).not.toBeInTheDocument();
    expect(heading.querySelector("span > svg")).toBeInTheDocument();
  });

  it("falls back to the dot for an unrecognized icon value instead of rendering nothing", () => {
    renderColumn({ column: { ...column, icon: "not-a-real-icon" } });
    const heading = screen.getByRole("heading", { level: 4 });
    expect(heading.querySelector("span > svg")).not.toBeInTheDocument();
    expect(heading.querySelector(".rounded-full.bg-info")).toBeInTheDocument();
  });
});
