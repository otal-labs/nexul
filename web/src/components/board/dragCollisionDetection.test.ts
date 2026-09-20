import type { ClientRect, DroppableContainer } from "@dnd-kit/core";
import { describe, expect, it } from "vitest";

import { boardCollisionDetection } from "@/components/board/dragCollisionDetection";
import type { DropTargetData } from "@/components/board/dragMove";

const rect = (left: number, top: number, width: number, height: number): ClientRect => ({
  left,
  top,
  width,
  height,
  right: left + width,
  bottom: top + height,
});

const container = (id: string, data: DropTargetData): DroppableContainer =>
  ({ id, key: id, disabled: false, data: { current: data }, node: { current: null }, rect: { current: null } }) as DroppableContainer;

// A lane spans the same area as the columns inside it (see SwimlaneSection),
// so most scenarios have the pointer/collisionRect over both at once.
const lane = container("lane-1", { type: "lane", categoryId: "c-1" });
const laneRect = rect(0, 0, 400, 200);
const column = container("column-1", { type: "column", statusId: "done", categoryId: "c-1" });
const columnRect = rect(0, 0, 100, 200);
const card = container("t-9", { type: "card", ticketId: "t-9", statusId: "done", categoryId: "c-1" });
const cardRect = rect(0, 0, 100, 40);

const droppableRects = new Map([
  ["lane-1", laneRect],
  ["column-1", columnRect],
  ["t-9", cardRect],
]);

describe("boardCollisionDetection", () => {
  it("prefers the column when the pointer is over both the lane and its column", () => {
    const result = boardCollisionDetection({
      active: { id: "t-1" } as never,
      collisionRect: rect(10, 10, 50, 50),
      droppableRects,
      droppableContainers: [lane, column],
      pointerCoordinates: { x: 50, y: 50 },
    });
    expect(result.map((c) => c.id)).toEqual(["column-1"]);
  });

  it("falls back to the lane when the pointer is only over the lane", () => {
    const result = boardCollisionDetection({
      active: { id: "t-1" } as never,
      collisionRect: rect(300, 10, 50, 50),
      droppableRects,
      droppableContainers: [lane, column],
      pointerCoordinates: { x: 320, y: 30 },
    });
    expect(result.map((c) => c.id)).toEqual(["lane-1"]);
  });

  it("prefers the card when the pointer is over the card, its column, and its lane at once", () => {
    const result = boardCollisionDetection({
      active: { id: "t-1" } as never,
      collisionRect: rect(10, 10, 20, 20),
      droppableRects,
      droppableContainers: [lane, column, card],
      pointerCoordinates: { x: 20, y: 20 },
    });
    expect(result.map((c) => c.id)).toEqual(["t-9"]);
  });

  it("prefers the column over the lane for keyboard drags (no pointer coordinates)", () => {
    const result = boardCollisionDetection({
      active: { id: "t-1" } as never,
      collisionRect: rect(0, 0, 100, 200),
      droppableRects,
      droppableContainers: [lane, column],
      pointerCoordinates: null,
    });
    expect(result.map((c) => c.id)).toEqual(["column-1"]);
  });

  it("only considers same-stage columns in the same lane when a column itself is dragged", () => {
    const sameStage = container("column-1", { type: "column", statusId: "done", categoryId: "c-1", kind: "done" });
    const otherLane = container("column-2", { type: "column", statusId: "done", categoryId: "c-2", kind: "done" });
    const otherStage = container("column-3", { type: "column", statusId: "review", categoryId: "c-1", kind: "review" });
    const rects = new Map([...droppableRects, ["column-2", rect(0, 0, 100, 200)] as const, ["column-3", rect(0, 0, 100, 200)] as const]);
    const result = boardCollisionDetection({
      active: { id: "column-0", data: { current: { type: "column", statusId: "closed", categoryId: "c-1", kind: "done" } } } as never,
      collisionRect: rect(5, 0, 100, 200),
      droppableRects: rects,
      droppableContainers: [lane, sameStage, card, otherLane, otherStage],
      pointerCoordinates: { x: 20, y: 20 },
    });
    expect(result.map((c) => c.id)).toEqual(["column-1"]);
  });

  it("ignores the dragged ticket's own card droppable, falling through to its column", () => {
    // dragPreview.ts relocates the dragged ticket's own card into whatever cell
    // it's hovering, so its id can show up as a candidate right where the
    // pointer is; that must never win or it reads as dropping onto itself.
    const ghost = container("t-1", { type: "card", ticketId: "t-1", statusId: "done", categoryId: "c-1" });
    const rects = new Map([...droppableRects, ["t-1", rect(0, 0, 100, 40)] as const]);
    const result = boardCollisionDetection({
      active: { id: "t-1" } as never,
      collisionRect: rect(10, 10, 20, 20),
      droppableRects: rects,
      droppableContainers: [lane, column, ghost],
      pointerCoordinates: { x: 20, y: 20 },
    });
    expect(result.map((c) => c.id)).toEqual(["column-1"]);
  });
});
