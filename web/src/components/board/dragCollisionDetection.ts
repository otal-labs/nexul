import { closestCenter, pointerWithin, rectIntersection, type CollisionDetection } from "@dnd-kit/core";

import type { DropTargetData } from "@/components/board/dragMove";

const dataOf = (collision: ReturnType<CollisionDetection>[number]) =>
  collision.data?.droppableContainer.data.current as DropTargetData | undefined;

const matching = (candidates: ReturnType<CollisionDetection>, type: DropTargetData["type"]) =>
  candidates.filter((collision) => dataOf(collision)?.type === type);

export const boardCollisionDetection: CollisionDetection = (args) => {
  const activeData = args.active.data?.current as DropTargetData | undefined;
  // A column only ever swaps with same-stage columns in its own lane; pointer detection would keep landing on the cards inside them.
  if (activeData?.type === "column") {
    return matching(closestCenter(args), "column").filter((collision) => {
      const data = dataOf(collision);
      return data?.type === "column" && data.categoryId === activeData.categoryId && data.kind === activeData.kind;
    });
  }

  // A pointer over a card collides with card, column, and lane; resolve most-specific (card) first.
  const candidates = (args.pointerCoordinates ? pointerWithin(args) : rectIntersection(args)).filter(
    // The dragged ticket's own card droppable rides along with it; without this a hover self-drops instead of moving.
    (collision) => collision.id !== args.active.id,
  );
  const cardMatches = matching(candidates, "card");
  if (cardMatches.length > 0) return cardMatches;
  const columnMatches = matching(candidates, "column");
  return columnMatches.length > 0 ? columnMatches : candidates;
};
