import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Swimlane } from "@/components/board/KanbanBoard";
import { SwimlaneSection } from "@/components/board/SwimlaneSection";
import { useBoardStore } from "@/stores/boardStore";
import type { BoardStatus } from "@/models/Status";
import type { Ticket, TicketStatus } from "@/models/Ticket";

// TicketCard (T10) fetches ticket types and label colors via TanStack Query, so every render
// needs a QueryClient ancestor; empty-array responses suffice since this file asserts on swimlane layout, not badge colors (see TicketCard.test.tsx).
vi.mock("@/api/client", () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

// F6: TicketCard navigates directly via useNavigate, not an onSelect callback through SwimlaneSection/KanbanColumn.
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => vi.fn() };
});

const columns: BoardStatus[] = [
  { id: "open", name: "Open", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" },
  { id: "done", name: "Done", position: 1, kind: "done", icon: "", created_at: "", updated_at: "" },
];

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

const lane = (categoryId: string | null, tickets: Ticket[]): Swimlane => ({
  key: "Sprint 1",
  label: "Sprint 1",
  categoryId,
  tickets,
});

const renderSection = (props: Partial<Parameters<typeof SwimlaneSection>[0]> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SwimlaneSection
        lane={lane("c-1", [ticket("t-1", "Fix login", "open")])}
        columns={columns}
        onAddTicket={() => {}}
        {...props}
      />
    </QueryClientProvider>,
  );
};

describe("SwimlaneSection", () => {
  // The board store is module-global and persisted; reset so a collapse in one test can't leak into the next.
  beforeEach(() => useBoardStore.setState({ collapsedLaneKeys: [] }));

  it("collapses and expands the columns from the whole header row", async () => {
    const user = userEvent.setup();
    renderSection();
    const toggle = screen.getByRole("button", { name: "Sprint 1 1 tickets" });
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("region", { name: "Open column in Sprint 1" })).not.toBeInTheDocument();

    await user.click(toggle);
    expect(screen.getByRole("region", { name: "Open column in Sprint 1" })).toBeInTheDocument();
  });

  it("renders the lane label and ticket count", () => {
    renderSection();
    expect(screen.getByText("Sprint 1")).toBeInTheDocument();
    expect(screen.getByText("1 tickets")).toBeInTheDocument();
  });

  it("renders one column per status", () => {
    renderSection();
    expect(screen.getByRole("region", { name: "Open column in Sprint 1" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Done column in Sprint 1" })).toBeInTheDocument();
  });

  it("splits lane tickets across columns by status", () => {
    renderSection({ lane: lane("c-1", [ticket("t-1", "Fix login", "open"), ticket("t-2", "Wire FTS", "done")]) });
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });

  it("renders tickets within a column sorted by position, not input order", () => {
    renderSection({
      lane: lane("c-1", [
        ticket("t-1", "Wire FTS", "open", 2),
        ticket("t-2", "Fix login", "open", 0),
        ticket("t-3", "Add tests", "open", 1),
      ]),
    });
    const titles = screen.getAllByText(/^(Fix login|Wire FTS|Add tests)$/).map((el) => el.textContent);
    expect(titles).toEqual(["Fix login", "Add tests", "Wire FTS"]);
  });
});
