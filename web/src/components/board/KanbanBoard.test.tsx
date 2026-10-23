import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useDndContext, type DndContextProps, type DragOverEvent, type DragStartEvent } from "@dnd-kit/core";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
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

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => vi.fn() };
});

// Every card and column here is draggable, as for a viewer who can write tickets and projects, unless a test narrows it.
const viewer = vi.hoisted(() => ({ holds: new Set(["tickets:write", "projects:write"]) }));
vi.mock("@/hooks/useActionState", () => ({
  useActionState: ({ permission }: { permission: string }) => ({ state: viewer.holds.has(permission) ? "live" : "hidden" }),
}));

// jsdom never lays anything out, so drags are driven through the handlers the board hands to DndContext.
const dnd = vi.hoisted(() => ({ props: null as DndContextProps | null }));
vi.mock("@dnd-kit/core", async () => {
  const actual = await vi.importActual<typeof import("@dnd-kit/core")>("@dnd-kit/core");
  return {
    ...actual,
    DndContext: (props: DndContextProps) => {
      dnd.props = props;
      return (
        <actual.DndContext {...props}>
          {props.children}
          <DropTargets />
        </actual.DndContext>
      );
    },
  };
});

// The drop targets collision detection may pick, which dnd-kit limits to the enabled ones.
const DropTargets = () => {
  const { droppableContainers } = useDndContext();
  return (
    <ul aria-label="Drop targets">
      {droppableContainers.getEnabled().map((container) => (
        <li key={container.id}>{container.id}</li>
      ))}
    </ul>
  );
};

const dropTargets = () =>
  within(screen.getByRole("list", { name: "Drop targets" }))
    .queryAllByRole("listitem")
    .map((item) => item.textContent ?? "");

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
  viewer.holds = new Set(["tickets:write", "projects:write"]);
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: [] });
});

describe("KanbanBoard", () => {
  // Each query observer arms its own stale timer on mount, so per-card subscriptions stall opening a big board.
  it("subscribes to the board's queries once, however many cards it shows", () => {
    const observersWith = (cards: number) => {
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      const tickets = Array.from({ length: cards }, (_, i) => ticket(`t-${i}`, `Ticket ${i}`, "open", "c-1"));
      const { unmount } = render(
        <QueryClientProvider client={client}>
          <KanbanBoard
            projectId="p-1"
            columns={columns}
            swimlanes={[lane("Sprint 1", "Sprint 1", "c-1", tickets)]}
            onDrop={() => {}}
            onReorderColumns={() => {}}
            onAddTicket={() => {}}
          />
        </QueryClientProvider>,
      );
      const observers = client.getQueryCache().getAll().reduce((sum, query) => sum + query.getObserversCount(), 0);
      unmount();
      return observers;
    };
    expect(observersWith(20)).toBe(observersWith(1));
  });

  it("renders a swimlane per category and an uncategorized one for tickets with none", () => {
    const withUncategorized = [
      ...swimlanes,
      lane("__uncategorized__", "Uncategorized", null, [ticket("t-3", "Quick capture", "open")]),
    ];
    renderBoard(
      <KanbanBoard
        projectId="p-1"
        columns={columns}
        swimlanes={withUncategorized}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    expect(screen.getByRole("region", { name: "Sprint 1 swimlane" })).toBeInTheDocument();
    expect(screen.getByText("Fix login")).toBeInTheDocument();
    expect(screen.getByText("Wire FTS")).toBeInTheDocument();
    expect(screen.getByText("Uncategorized")).toBeInTheDocument();
    expect(screen.getByText("Quick capture")).toBeInTheDocument();
  });

  it("ignores an over change that arrives with the pointer still, so the ghost's own layout shift cannot flip it between two cells", () => {
    const twoLanes = [
      lane("Sprint 1", "Sprint 1", "c-1", [ticket("t-1", "Fix login", "open", "c-1")]),
      lane("Sprint 2", "Sprint 2", "c-2", [ticket("t-3", "Ship search", "open", "c-2")]),
    ];
    renderBoard(
      <KanbanBoard
        projectId="p-1"
        columns={columns}
        swimlanes={twoLanes}
        onDrop={() => {}}
        onReorderColumns={() => {}}
        onAddTicket={() => {}}
      />,
    );
    const active = { id: "t-1", data: { current: undefined }, rect: { current: { initial: null, translated: null } } };
    const overDone = (categoryId: string, delta: { x: number; y: number }) =>
      ({
        active,
        delta,
        collisions: null,
        activatorEvent: null,
        over: { id: `done-${categoryId}`, rect: null, disabled: false, data: { current: { type: "column", statusId: "done", categoryId, kind: "done" } } },
      }) as unknown as DragOverEvent;
    const doneIn = (laneLabel: string) => within(screen.getByRole("region", { name: `Done column in ${laneLabel}` }));

    act(() => dnd.props!.onDragStart!({ active } as unknown as DragStartEvent));
    act(() => dnd.props!.onDragOver!(overDone("c-1", { x: 40, y: 10 })));
    expect(doneIn("Sprint 1").getByText("Fix login")).toBeInTheDocument();

    act(() => dnd.props!.onDragOver!(overDone("c-2", { x: 40, y: 10 })));
    expect(doneIn("Sprint 1").getByText("Fix login")).toBeInTheDocument();
    expect(doneIn("Sprint 2").queryByText("Fix login")).not.toBeInTheDocument();

    act(() => dnd.props!.onDragOver!(overDone("c-2", { x: 40, y: 90 })));
    expect(doneIn("Sprint 2").getByText("Fix login")).toBeInTheDocument();
  });

  it("renders the dragged card's overlay outside the page, so a blurred page frame cannot offset it from the pointer", async () => {
    const { container } = renderBoard(
      <KanbanBoard projectId="p-1" columns={columns} swimlanes={swimlanes} onDrop={() => {}} onReorderColumns={() => {}} onAddTicket={() => {}} />,
    );
    const card = within(container).getByText("Fix login").closest<HTMLElement>('[role="button"]')!;
    fireEvent.mouseDown(card, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(document, { clientX: 20, clientY: 0 });

    const copies = await screen.findAllByText("Fix login");
    expect(copies.filter((copy) => !container.contains(copy))).toHaveLength(1);
    fireEvent.mouseUp(document);
  });

  it("picks a card up with Space, so a keyboard user can move it", async () => {
    const { container } = renderBoard(
      <KanbanBoard projectId="p-1" columns={columns} swimlanes={swimlanes} onDrop={() => {}} onReorderColumns={() => {}} onAddTicket={() => {}} />,
    );
    const card = within(container).getByText("Fix login").closest<HTMLElement>('[role="button"]')!;
    card.focus();
    fireEvent.keyDown(card, { code: "Space", key: " " });

    const copies = await screen.findAllByText("Fix login");
    expect(copies.filter((copy) => !container.contains(copy))).toHaveLength(1);
  });

  it("takes no drop in a collapsed lane, whose hidden columns still measure over the lane below", async () => {
    const twoLanes = [
      lane("Sprint 2", "Sprint 2", "c-2", [ticket("t-3", "Ship search", "open", "c-2")]),
      lane("Sprint 1", "Sprint 1", "c-1", [ticket("t-1", "Fix login", "open", "c-1")]),
    ];
    renderBoard(<KanbanBoard projectId="p-1" columns={columns} swimlanes={twoLanes} onDrop={() => {}} onReorderColumns={() => {}} onAddTicket={() => {}} />);
    await userEvent.click(within(screen.getByRole("region", { name: "Sprint 2 swimlane" })).getByRole("button", { expanded: true }));

    const targets = dropTargets();
    expect(targets.filter((id) => id.startsWith("column-Sprint 2-") || id === "t-3")).toEqual([]);
    expect(targets).toEqual(expect.arrayContaining(["column-Sprint 1-open", "column-Sprint 1-done", "t-1"]));
  });

  it("still takes a card in a column the viewer may not reorder", () => {
    viewer.holds = new Set(["tickets:write"]);
    renderBoard(<KanbanBoard projectId="p-1" columns={columns} swimlanes={swimlanes} onDrop={() => {}} onReorderColumns={() => {}} onAddTicket={() => {}} />);

    expect(screen.queryByRole("button", { name: "Reorder Done" })).not.toBeInTheDocument();
    expect(dropTargets()).toEqual(expect.arrayContaining(["column-Sprint 1-open", "column-Sprint 1-in_progress", "column-Sprint 1-done"]));
  });
});
