import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";

import { api } from "@/api/client";
import { KanbanBoard, type Swimlane } from "@/components/board/KanbanBoard";
import type { BoardStatus } from "@/models/Status";
import type { Ticket, TicketStatus } from "@/models/Ticket";

// TicketCard fetches ticket types and label colors via TanStack Query hooks,
// so every render needs a QueryClient ancestor. Empty-array responses are
// enough here since this file asserts on drag/drop and layout, not badge
// colors (see TicketCard.test.tsx for that).
vi.mock("@/api/client", () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

// TicketCard navigates directly via useNavigate rather than an onSelect
// callback bubbled up through KanbanBoard/SwimlaneSection/KanbanColumn.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => mockNavigate };
});

const renderBoard = (ui: ReactElement) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
};

const ticket = (id: string, title: string, status: string, categoryId: string = ""): Ticket => ({
  id,
  project_id: "p-1",
  category_id: categoryId,
  type_id: "ticket-type-task",
  title,
  body: "",
  status: status as TicketStatus,
  position: 0,
  number: 1,
  doc_id: "",
  developer: "onik97",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
});

const columns: BoardStatus[] = [
  { id: "open", name: "Open", position: 0, kind: "progress", icon: "", created_at: "", updated_at: "" },
  { id: "in_progress", name: "In progress", position: 1, kind: "progress", icon: "", created_at: "", updated_at: "" },
  { id: "done", name: "Done", position: 2, kind: "done", icon: "", created_at: "", updated_at: "" },
];

const lane = (key: string, label: string, categoryId: string | null, tickets: Ticket[]): Swimlane => ({
  key,
  label,
  categoryId,
  tickets,
});

const swimlanes = [
  lane("Sprint 1", "Sprint 1", "c-1", [ticket("t-1", "Fix login", "open", "c-1"), ticket("t-2", "Wire FTS", "done", "c-1")]),
];

beforeEach(() => {
  mockNavigate.mockReset();
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: [] });
});

describe("KanbanBoard", () => {
  it("renders swimlanes with all status columns", () => {
    renderBoard(
      <KanbanBoard
        columns={columns}
        swimlanes={swimlanes}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    expect(screen.getByRole("region", { name: "Sprint 1 swimlane" })).toBeInTheDocument();
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
  });

  it("renders an uncategorized section", () => {
    const withUncategorized = [
      ...swimlanes,
      lane("__uncategorized__", "Uncategorized", null, [ticket("t-3", "Quick capture", "open")]),
    ];
    renderBoard(
      <KanbanBoard
        columns={columns}
        swimlanes={withUncategorized}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    expect(screen.getByText("Uncategorized")).toBeInTheDocument();
    expect(screen.getByText("Quick capture")).toBeInTheDocument();
  });

  it("shows the human-readable project-prefixed ticket id on cards", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" } };
      return { data: [] };
    });
    renderBoard(
      <KanbanBoard
        columns={columns}
        swimlanes={swimlanes}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    expect((await screen.findAllByText("BE-1", { exact: false })).length).toBeGreaterThan(0);
  });

  it("navigates to the ticket detail page on click", async () => {
    const user = userEvent.setup();
    renderBoard(
      <KanbanBoard
        columns={columns}
        swimlanes={swimlanes}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    await user.click(screen.getByText("Fix login"));
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/t-1");
  });
});
