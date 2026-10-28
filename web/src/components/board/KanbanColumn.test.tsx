import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardCardProvider } from "@/components/board/BoardCardProvider";
import { KanbanColumn } from "@/components/board/KanbanColumn";
import type { BoardStatus } from "@/models/Status";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";
import type { Ticket, TicketStatus } from "@/models/Ticket";

// TicketCard (T10) fetches ticket types and label colors via TanStack Query, so every render
// needs a QueryClient ancestor; empty-array responses suffice since this file doesn't assert on badge colors (see TicketCard.test.tsx).
vi.mock("@/api/client", async () => {
  const { asOwner } = await import("@/test/viewer");
  return { api: { get: vi.fn(asOwner()), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() };
});

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
  developer_id: "",
  tester_id: "",
  reporter: { kind: "user", login: "onik97" },
  due_on: "",
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
});

const renderColumn = (props: Partial<Parameters<typeof KanbanColumn>[0]> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <BoardCardProvider projectId="p-1">
        <KanbanColumn
          droppableId="column-Sprint 1-open"
          laneLabel="Sprint 1"
          categoryId="c-1"
          column={column}
          collapsed={false}
          tickets={[ticket("t-1", "Fix login", "open")]}
          onAddTicket={() => {}}
          {...props}
        />
      </BoardCardProvider>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  mockNavigate.mockReset();
});

describe("KanbanColumn", () => {
  it("names the column, counts its tickets and shows a card for each", () => {
    renderColumn({ tickets: [ticket("t-1", "Fix login", "open"), ticket("t-2", "Wire FTS", "open")] });
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });

  it("mounts a long column 10 cards at a time, counting every ticket in the header", async () => {
    const user = userEvent.setup();
    const many = Array.from({ length: 25 }, (_, i) => ticket(`t-${i}`, `Ticket ${i}`, "open", i));
    renderColumn({ tickets: many });
    expect(screen.getByLabelText("25 tickets")).toBeInTheDocument();
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(10);

    await user.click(screen.getByRole("button", { name: "Show 10 more of 15" }));
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(20);
    await user.click(screen.getByRole("button", { name: "Show 5 more of 5" }));
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(25);
    expect(screen.queryByRole("button", { name: /^Show \d+ more/ })).not.toBeInTheDocument();
  });

  it("folds an empty column to a narrow strip that still names it and says it is empty", () => {
    renderColumn({ tickets: [] });
    expect(screen.getByRole("region", { name: "Open column in Sprint 1" })).toHaveClass("w-10");
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("No tickets")).toHaveClass("sr-only");
  });

  it("gives a column with cards the full width", () => {
    renderColumn();
    expect(screen.getByRole("region", { name: "Open column in Sprint 1" })).not.toHaveClass("w-10");
  });

  it("navigates to the ticket detail page on click", async () => {
    const user = userEvent.setup();
    renderColumn();
    await user.click(screen.getByText("Fix login"));
    expect(mockNavigate).toHaveBeenCalledWith("/acme/tickets/t-1");
  });

  it("calls onAddTicket with this column's status id when the + button is clicked", async () => {
    const user = userEvent.setup();
    const onAddTicket = vi.fn();
    renderColumn({ onAddTicket });
    await user.click(await screen.findByRole("button", { name: "New ticket in Open" }));
    expect(onAddTicket).toHaveBeenCalledWith("open");
  });

  it("makes the header's name and count the column's drag surface, leaving the add button outside it", async () => {
    renderColumn();
    const surface = await screen.findByRole("button", { name: "Reorder Open" });
    const add = screen.getByRole("button", { name: "New ticket in Open" });
    expect(surface).toHaveAttribute("aria-roledescription", "sortable");
    expect(surface).not.toBe(add);
    expect(surface).not.toContainElement(add);
    expect(surface).toHaveTextContent("Open");
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

  describe("for a member who may read the board and not write it", () => {
    const readOnly = (showLocked: boolean) => vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:read", "projects:read"], showLocked)));

    it("shows the column and its cards, which open but cannot be dragged, with no control that writes", async () => {
      readOnly(false);
      const user = userEvent.setup();
      renderColumn();

      const card = (await screen.findByText("Fix login")).closest("[aria-disabled]");
      expect(card).toHaveAttribute("aria-disabled", "true");
      expect(screen.queryByRole("button", { name: "Reorder Open" })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "New ticket in Open" })).not.toBeInTheDocument();
      await user.click(screen.getByText("Fix login"));
      expect(mockNavigate).toHaveBeenCalledWith("/acme/tickets/t-1");
    });

    it("locks the add button and offers no way to drag the column when a role shows locked actions", async () => {
      readOnly(true);
      const user = userEvent.setup();
      const onAddTicket = vi.fn();
      renderColumn({ onAddTicket });

      const add = await screen.findByRole("button", { name: "New ticket in Open" });
      expect(add).toHaveAttribute("aria-disabled", "true");
      expect(screen.queryByRole("button", { name: "Reorder Open" })).not.toBeInTheDocument();
      await user.click(add);
      expect(onAddTicket).not.toHaveBeenCalled();
    });
  });
});
