import type { ClientRect, DroppableContainer, SensorContext } from "@dnd-kit/core";
import { describe, expect, it, vi } from "vitest";

import { boardKeyboardCoordinates } from "@/components/board/dragKeyboardCoordinates";
import type { DropTargetData } from "@/components/board/dragMove";

const rect = (left: number, top: number, width: number, height: number): ClientRect => ({
  left,
  top,
  width,
  height,
  right: left + width,
  bottom: top + height,
});

const container = (id: string, data?: DropTargetData): DroppableContainer =>
  ({ id, key: id, disabled: false, data: { current: data }, node: { current: null }, rect: { current: null } }) as DroppableContainer;

const buildContext = (containers: DroppableContainer[], rects: Map<string, ClientRect>): SensorContext =>
  ({
    active: { id: "t-1" },
    collisionRect: rect(0, 100, 100, 50),
    droppableRects: rects,
    droppableContainers: { getEnabled: () => containers },
  }) as unknown as SensorContext;

const keyEvent = (code: string) => ({ code, preventDefault: vi.fn() }) as unknown as KeyboardEvent & { preventDefault: () => void };

describe("boardKeyboardCoordinates", () => {
  it("jumps to the nearest droppable below on ArrowDown", () => {
    const below = container("below");
    const above = container("above");
    const rects = new Map([
      ["below", rect(0, 300, 100, 50)],
      ["above", rect(0, -100, 100, 50)],
    ]);
    const context = buildContext([below, above], rects);
    const event = keyEvent("ArrowDown");

    const result = boardKeyboardCoordinates(event, { active: "t-1", currentCoordinates: { x: 0, y: 100 }, context });

    expect(result).toEqual({ x: 0, y: 300 });
    expect(event.preventDefault).toHaveBeenCalled();
  });

  it("jumps to the nearest droppable above on ArrowUp", () => {
    const below = container("below");
    const above = container("above");
    const rects = new Map([
      ["below", rect(0, 300, 100, 50)],
      ["above", rect(0, -100, 100, 50)],
    ]);
    const context = buildContext([below, above], rects);

    const result = boardKeyboardCoordinates(keyEvent("ArrowUp"), {
      active: "t-1",
      currentCoordinates: { x: 0, y: 100 },
      context,
    });

    expect(result).toEqual({ x: 0, y: -100 });
  });

  it("ignores non-arrow keys", () => {
    const context = buildContext([container("a")], new Map([["a", rect(0, 300, 100, 50)]]));
    const event = keyEvent("Space");

    const result = boardKeyboardCoordinates(event, { active: "t-1", currentCoordinates: { x: 0, y: 100 }, context });

    expect(result).toBeUndefined();
    expect(event.preventDefault).not.toHaveBeenCalled();
  });

  it("skips a closer card droppable and jumps to the next column/lane instead", () => {
    // A card sits much closer (it's inside the same column) than the next
    // column/lane, but keyboard drags should keep the ticket-01 column/lane
    // jump behaviour — cards are pointer/touch reorder targets only.
    const nearCard = container("near-card", { type: "card", ticketId: "t-9", statusId: "open", categoryId: "c-1" });
    const farColumn = container("far-column", { type: "column", statusId: "done", categoryId: "c-1" });
    const rects = new Map([
      ["near-card", rect(0, 160, 100, 50)],
      ["far-column", rect(0, 300, 100, 50)],
    ]);
    const context = buildContext([nearCard, farColumn], rects);

    const result = boardKeyboardCoordinates(keyEvent("ArrowDown"), {
      active: "t-1",
      currentCoordinates: { x: 0, y: 100 },
      context,
    });

    expect(result).toEqual({ x: 0, y: 300 });
  });

  it("returns undefined when no droppable exists in the pressed direction", () => {
    const context = buildContext([container("below")], new Map([["below", rect(0, 300, 100, 50)]]));

    const result = boardKeyboardCoordinates(keyEvent("ArrowUp"), {
      active: "t-1",
      currentCoordinates: { x: 0, y: 100 },
      context,
    });

    expect(result).toBeUndefined();
  });
});
