import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
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
          folded={false}
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
  it("shows a card for each ticket", () => {
    renderColumn({ tickets: [ticket("t-1", "Fix login", "open"), ticket("t-2", "Wire FTS", "open")] });
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });

  it("mounts a long cell 10 cards at a time", async () => {
    const user = userEvent.setup();
    const many = Array.from({ length: 25 }, (_, i) => ticket(`t-${i}`, `Ticket ${i}`, "open", i));
    renderColumn({ tickets: many });
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(10);

    await user.click(screen.getByRole("button", { name: "Show 10 more of 15" }));
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(20);
    await user.click(screen.getByRole("button", { name: "Show 5 more of 5" }));
    expect(screen.getAllByText(/^Ticket \d+$/)).toHaveLength(25);
    expect(screen.queryByRole("button", { name: /^Show \d+ more/ })).not.toBeInTheDocument();
  });

  it("says an empty cell has no tickets", () => {
    renderColumn({ tickets: [] });
    const region = screen.getByRole("region", { name: "Open column in Sprint 1" });
    expect(within(region).getByText("No tickets")).toBeInTheDocument();
  });

  it("shows a folded cell as its ticket count and mounts no card", () => {
    renderColumn({ folded: true, tickets: [ticket("t-1", "Fix login", "open"), ticket("t-2", "Wire FTS", "open")] });
    expect(screen.getByText("2 tickets, folded")).toBeInTheDocument();
    expect(screen.queryByText("Fix login")).not.toBeInTheDocument();
  });

  it("opens a card's ticket on click", async () => {
    const user = userEvent.setup();
    renderColumn();
    await user.click(screen.getByText("Fix login"));
    expect(mockNavigate).toHaveBeenCalledWith("/acme/tickets/t-1");
  });

  describe("for a member who may read the board and not write it", () => {
    const readOnly = (showLocked: boolean) => vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:read", "projects:read"], showLocked)));

    it("shows cards that open but cannot be dragged", async () => {
      readOnly(false);
      const user = userEvent.setup();
      renderColumn();

      const card = (await screen.findByText("Fix login")).closest("[aria-disabled]");
      expect(card).toHaveAttribute("aria-disabled", "true");
      await user.click(screen.getByText("Fix login"));
      expect(mockNavigate).toHaveBeenCalledWith("/acme/tickets/t-1");
    });
  });
});
