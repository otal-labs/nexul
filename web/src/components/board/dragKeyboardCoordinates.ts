import { closestCorners, getFirstCollision, type KeyboardCoordinateGetter } from "@dnd-kit/core";

import type { DropTargetData } from "@/components/board/dragMove";

// Jumps straight to the nearest droppable in the pressed direction, since the default per-pixel getter is too slow here.
const DIRECTIONS: Record<string, "up" | "down" | "left" | "right"> = {
  ArrowDown: "down",
  ArrowUp: "up",
  ArrowLeft: "left",
  ArrowRight: "right",
};

export const boardKeyboardCoordinates: KeyboardCoordinateGetter = (event, { context }) => {
  const direction = DIRECTIONS[event.code];
  if (!direction) return undefined;

  const { active, collisionRect, droppableRects, droppableContainers } = context;
  if (!active || !collisionRect) return undefined;

  event.preventDefault();

  const candidates = droppableContainers.getEnabled().filter((container) => {
    // Cards excluded so a nearby one can't hijack a jump meant for the next column/lane; keyboard reorder isn't in scope.
    if ((container.data.current as DropTargetData | undefined)?.type === "card") return false;
    const rect = droppableRects.get(container.id);
    if (!rect) return false;
    if (direction === "down") return rect.top > collisionRect.top;
    if (direction === "up") return rect.top < collisionRect.top;
    if (direction === "left") return rect.left < collisionRect.left;
    return rect.left > collisionRect.left;
  });

  const closestId = getFirstCollision(
    closestCorners({ active, collisionRect, droppableRects, droppableContainers: candidates, pointerCoordinates: null }),
    "id",
  );
  if (closestId == null) return undefined;

  const rect = droppableRects.get(closestId);
  return rect ? { x: rect.left, y: rect.top } : undefined;
};
