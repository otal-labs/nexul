import { describe, expect, it } from "vitest";

import { applyDragPreview, applyDropResult, resolveDragCell, sameDragCell } from "@/components/board/dragPreview";
import type { Swimlane } from "@/components/board/KanbanBoard";
import type { Ticket } from "@/models/Ticket";

const ticket = (id: string, status: string, categoryId: string, position: number): Ticket => ({
  id,
  project_id: "p-1",
  category_id: categoryId,
  type_id: "ticket-type-task",
  title: id,
  body: "",
  status: status as Ticket["status"],
  position,
  number: 1,
  doc_id: "",
  assignee: "",
  created_at: "",
  updated_at: "",
  labels: [],
});

const lane = (key: string, categoryId: string | null, tickets: Ticket[]): Swimlane => ({
  key,
  label: key,
  categoryId,
  tickets,
});

describe("resolveDragCell", () => {
  const active = { status: "open", categoryId: "c-1" };

  it("returns null with no drop target", () => {
    expect(resolveDragCell(active, undefined)).toBeNull();
  });

  it("takes status and category straight from a column target", () => {
    expect(resolveDragCell(active, { type: "column", statusId: "done", categoryId: "c-2" })).toEqual({
      status: "done",
      categoryId: "c-2",
    });
  });

  it("takes status and category from a card target and remembers which side of it to land on", () => {
    expect(resolveDragCell(active, { type: "card", ticketId: "t-9", statusId: "done", categoryId: "c-2" }, true)).toEqual({
      status: "done",
      categoryId: "c-2",
      anchor: { ticketId: "t-9", after: true },
    });
  });

  it("keeps the active ticket's own status for a lane target — a lane has no column under it", () => {
    expect(resolveDragCell(active, { type: "lane", categoryId: "c-2" })).toEqual({ status: "open", categoryId: "c-2" });
  });
});

describe("sameDragCell", () => {
  it("treats two nulls as the same cell", () => {
    expect(sameDragCell(null, null)).toBe(true);
  });

  it("treats matching status+category as the same cell even as different objects", () => {
    expect(sameDragCell({ status: "open", categoryId: "c-1" }, { status: "open", categoryId: "c-1" })).toBe(true);
  });

  it("ignores the anchor — within a cell dnd-kit previews on its own", () => {
    expect(
      sameDragCell(
        { status: "open", categoryId: "c-1", anchor: { ticketId: "t-2", after: false } },
        { status: "open", categoryId: "c-1", anchor: { ticketId: "t-3", after: true } },
      ),
    ).toBe(true);
  });

  it("treats a differing status or category as a different cell", () => {
    expect(sameDragCell({ status: "open", categoryId: "c-1" }, { status: "done", categoryId: "c-1" })).toBe(false);
    expect(sameDragCell({ status: "open", categoryId: "c-1" }, { status: "open", categoryId: "c-2" })).toBe(false);
    expect(sameDragCell({ status: "open", categoryId: "c-1" }, null)).toBe(false);
  });
});

describe("applyDragPreview", () => {
  it("leaves swimlanes untouched with no target cell", () => {
    const swimlanes = [lane("A", "c-1", [ticket("t-1", "open", "c-1", 0)])];
    expect(applyDragPreview(swimlanes, "t-1", null)).toBe(swimlanes);
  });

  it("leaves swimlanes untouched when the target cell is the ticket's own — dnd-kit's own SortableContext already previews that", () => {
    const swimlanes = [lane("A", "c-1", [ticket("t-1", "open", "c-1", 0), ticket("t-2", "open", "c-1", 1)])];
    expect(applyDragPreview(swimlanes, "t-1", { status: "open", categoryId: "c-1" })).toBe(swimlanes);
  });

  it("moves the ticket to the end of a different column in the same swimlane", () => {
    const swimlanes = [
      lane("A", "c-1", [ticket("t-1", "open", "c-1", 0), ticket("t-2", "in_progress", "c-1", 0), ticket("t-3", "in_progress", "c-1", 1)]),
    ];
    const preview = applyDragPreview(swimlanes, "t-1", { status: "in_progress", categoryId: "c-1" });
    const inProgress = preview[0]!.tickets.filter((t) => t.status === "in_progress").sort((a, b) => a.position - b.position);
    expect(inProgress.map((t) => t.id)).toEqual(["t-2", "t-3", "t-1"]);
  });

  it("slides the ticket in above or below the anchored card, renumbering the cell", () => {
    const swimlanes = [
      lane("A", "c-1", [ticket("t-1", "open", "c-1", 0), ticket("t-2", "done", "c-1", 0), ticket("t-3", "done", "c-1", 1)]),
    ];
    const order = (anchor: { ticketId: string; after: boolean }) =>
      applyDragPreview(swimlanes, "t-1", { status: "done", categoryId: "c-1", anchor })[0]!
        .tickets.filter((t) => t.status === "done")
        .sort((a, b) => a.position - b.position)
        .map((t) => `${t.id}@${t.position}`);
    expect(order({ ticketId: "t-2", after: false })).toEqual(["t-1@0", "t-2@1", "t-3@2"]);
    expect(order({ ticketId: "t-2", after: true })).toEqual(["t-2@0", "t-1@1", "t-3@2"]);
    expect(order({ ticketId: "t-gone", after: true })).toEqual(["t-2@0", "t-3@1", "t-1@2"]);
  });

  it("moves the ticket into an empty target column", () => {
    const swimlanes = [lane("A", "c-1", [ticket("t-1", "open", "c-1", 0)])];
    const preview = applyDragPreview(swimlanes, "t-1", { status: "done", categoryId: "c-1" });
    expect(preview[0]!.tickets.map((t) => ({ id: t.id, status: t.status }))).toEqual([{ id: "t-1", status: "done" }]);
  });

  it("moves the ticket into a different swimlane, matching the target's category", () => {
    const swimlanes = [
      lane("A", "c-1", [ticket("t-1", "open", "c-1", 0)]),
      lane("B", "c-2", [ticket("t-2", "open", "c-2", 0)]),
    ];
    const preview = applyDragPreview(swimlanes, "t-1", { status: "open", categoryId: "c-2" });
    expect(preview[0]!.tickets.map((t) => t.id)).toEqual([]);
    expect(preview[1]!.tickets.sort((a, b) => a.position - b.position).map((t) => t.id)).toEqual(["t-2", "t-1"]);
    expect(preview[1]!.tickets.find((t) => t.id === "t-1")?.category_id).toBe("c-2");
  });
});

describe("applyDropResult", () => {
  it("moves the ticket to its new status and swimlane and applies every position update", () => {
    const swimlanes = [
      lane("A", "c-1", [ticket("t-1", "open", "c-1", 0), ticket("t-2", "open", "c-1", 1)]),
      lane("B", "c-2", [ticket("t-3", "done", "c-2", 0)]),
    ];
    const result = applyDropResult(swimlanes, "t-1", [
      { kind: "status", ticketId: "t-1", status: "done" },
      { kind: "category", ticketId: "t-1", categoryId: "c-2" },
      { kind: "reorder", updates: [{ ticketId: "t-1", position: 0 }, { ticketId: "t-3", position: 1 }] },
    ]);
    expect(result[0]!.tickets.map((t) => t.id)).toEqual(["t-2"]);
    expect(result[1]!.tickets.map((t) => `${t.id}:${t.status}:${t.position}`)).toEqual(["t-3:done:1", "t-1:done:0"]);
  });

  it("returns the same array when there is nothing to apply", () => {
    const swimlanes = [lane("A", "c-1", [ticket("t-1", "open", "c-1", 0)])];
    expect(applyDropResult(swimlanes, "t-1", [])).toBe(swimlanes);
  });
});
