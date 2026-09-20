import { describe, expect, it } from "vitest";

import { resolveDragMove, ticketsInCell, type DragTicket } from "@/components/board/dragMove";

const active: DragTicket = { id: "t-1", status: "open", categoryId: "c-1", position: 1 };

describe("ticketsInCell", () => {
  it("keeps only tickets matching both status and categoryId, sorted by position", () => {
    const tickets: DragTicket[] = [
      { id: "t-3", status: "open", categoryId: "c-1", position: 2 },
      { id: "t-wrong-status", status: "done", categoryId: "c-1", position: 0 },
      { id: "t-1", status: "open", categoryId: "c-1", position: 0 },
      // Same-named categories from different projects merge into one swimlane,
      // so a ticket sharing the status but not the real categoryId must not
      // leak into the cell.
      { id: "t-wrong-category", status: "open", categoryId: "c-9", position: 1 },
    ];
    expect(ticketsInCell(tickets, "open", "c-1")).toEqual([
      { id: "t-1", status: "open", categoryId: "c-1", position: 0 },
      { id: "t-3", status: "open", categoryId: "c-1", position: 2 },
    ]);
  });
});

describe("resolveDragMove", () => {
  it("maps a column drop in the same swimlane to a status move, landing at the end of the empty cell", () => {
    expect(resolveDragMove(active, { type: "column", statusId: "done", categoryId: "c-1" }, [])).toEqual([
      { kind: "status", ticketId: "t-1", status: "done" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }] },
    ]);
  });

  it("keeps the slot the ghost was previewed into on a column drop", () => {
    const cellTickets: DragTicket[] = [
      { id: "t-8", status: "done", categoryId: "c-1", position: 0 },
      active,
      { id: "t-9", status: "done", categoryId: "c-1", position: 1 },
    ];
    expect(resolveDragMove(active, { type: "column", statusId: "done", categoryId: "c-1" }, cellTickets)).toEqual([
      { kind: "status", ticketId: "t-1", status: "done" },
      {
        kind: "reorder",
        // t-1 is included even though its slot index equals its old position: the status change re-appends it server-side.
        updates: [
          { ticketId: "t-1", position: 1 },
          { ticketId: "t-9", position: 2 },
        ],
      },
    ]);
  });

  it("maps a column drop in a different swimlane to both a status and a category move", () => {
    expect(resolveDragMove(active, { type: "column", statusId: "done", categoryId: "c-9" }, [])).toEqual([
      { kind: "status", ticketId: "t-1", status: "done" },
      { kind: "category", ticketId: "t-1", categoryId: "c-9" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }] },
    ]);
  });

  it("maps a lane drop to a category move", () => {
    expect(resolveDragMove(active, { type: "lane", categoryId: "c-2" }, [])).toEqual([
      { kind: "category", ticketId: "t-1", categoryId: "c-2" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }] },
    ]);
  });

  it("ignores a drop with no resolved target", () => {
    expect(resolveDragMove(active, undefined, [])).toEqual([]);
  });

  it("maps a card drop in a different column to a status move, inserted at that card's slot", () => {
    const target = { type: "card" as const, ticketId: "t-9", statusId: "done", categoryId: "c-1" };
    const cellTickets: DragTicket[] = [
      { id: "t-8", status: "done", categoryId: "c-1", position: 0 },
      { id: "t-9", status: "done", categoryId: "c-1", position: 1 },
    ];
    expect(resolveDragMove(active, target, cellTickets)).toEqual([
      { kind: "status", ticketId: "t-1", status: "done" },
      {
        kind: "reorder",
        updates: [
          { ticketId: "t-1", position: 1 },
          { ticketId: "t-9", position: 2 },
        ],
      },
    ]);
  });

  it("maps a card drop in the same column but a different swimlane to a category move — dropping among that swimlane's cards moves it there, it doesn't snap back", () => {
    const target = { type: "card" as const, ticketId: "t-9", statusId: "open", categoryId: "c-9" };
    expect(resolveDragMove(active, target, [])).toEqual([
      { kind: "category", ticketId: "t-1", categoryId: "c-9" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }] },
    ]);
  });

  it("maps a card drop in a different column and a different swimlane to both moves at once", () => {
    const target = { type: "card" as const, ticketId: "t-9", statusId: "done", categoryId: "c-9" };
    expect(resolveDragMove(active, target, [])).toEqual([
      { kind: "status", ticketId: "t-1", status: "done" },
      { kind: "category", ticketId: "t-1", categoryId: "c-9" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }] },
    ]);
  });

  it("ignores dropping a card onto itself", () => {
    const target = { type: "card" as const, ticketId: "t-1", statusId: "open", categoryId: "c-1" };
    expect(resolveDragMove(active, target, [])).toEqual([]);
  });

  it("reorders within the same cell — moving down past a sibling", () => {
    const columnTickets: DragTicket[] = [
      { id: "t-1", status: "open", categoryId: "c-1", position: 0 },
      { id: "t-2", status: "open", categoryId: "c-1", position: 1 },
      { id: "t-3", status: "open", categoryId: "c-1", position: 2 },
    ];
    const draggedActive: DragTicket = columnTickets[0]!;
    const target = { type: "card" as const, ticketId: "t-2", statusId: "open", categoryId: "c-1" };
    expect(resolveDragMove(draggedActive, target, columnTickets)).toEqual([
      {
        kind: "reorder",
        updates: [
          { ticketId: "t-2", position: 0 },
          { ticketId: "t-1", position: 1 },
        ],
      },
    ]);
  });

  it("reorders within the same cell — moving up past a sibling", () => {
    const columnTickets: DragTicket[] = [
      { id: "t-1", status: "open", categoryId: "c-1", position: 0 },
      { id: "t-2", status: "open", categoryId: "c-1", position: 1 },
      { id: "t-3", status: "open", categoryId: "c-1", position: 2 },
    ];
    const draggedActive: DragTicket = columnTickets[2]!;
    const target = { type: "card" as const, ticketId: "t-1", statusId: "open", categoryId: "c-1" };
    expect(resolveDragMove(draggedActive, target, columnTickets)).toEqual([
      {
        kind: "reorder",
        updates: [
          { ticketId: "t-3", position: 0 },
          { ticketId: "t-1", position: 1 },
          { ticketId: "t-2", position: 2 },
        ],
      },
    ]);
  });

  it("closes gaps in the source positions — the backend never renumbers on its own", () => {
    const columnTickets: DragTicket[] = [
      { id: "t-1", status: "open", categoryId: "c-1", position: 0 },
      { id: "t-2", status: "open", categoryId: "c-1", position: 5 },
      { id: "t-3", status: "open", categoryId: "c-1", position: 9 },
    ];
    const draggedActive: DragTicket = columnTickets[0]!;
    const target = { type: "card" as const, ticketId: "t-3", statusId: "open", categoryId: "c-1" };
    expect(resolveDragMove(draggedActive, target, columnTickets)).toEqual([
      {
        kind: "reorder",
        updates: [
          { ticketId: "t-2", position: 0 },
          { ticketId: "t-3", position: 1 },
          { ticketId: "t-1", position: 2 },
        ],
      },
    ]);
  });

  it("does nothing on a same-cell drop over the column itself — the ghost already sits where it started", () => {
    const columnTickets: DragTicket[] = [
      { id: "t-2", status: "open", categoryId: "c-1", position: 0 },
      active,
    ];
    expect(resolveDragMove(active, { type: "column", statusId: "open", categoryId: "c-1" }, columnTickets)).toEqual([]);
  });
});
