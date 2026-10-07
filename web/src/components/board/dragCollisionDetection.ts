import { closestCenter, pointerWithin, rectIntersection, type CollisionDetection, type DroppableContainer } from "@dnd-kit/core";

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

  // The dragged ticket's own card droppable rides along with it; without this a hover self-drops instead of moving.
  const others = args.droppableContainers.filter((container) => container.id !== args.active.id);
  const isCard = (container: DroppableContainer) => (container.data.current as DropTargetData | undefined)?.type === "card";

  // A pointer over a card collides with card, column, and lane; resolve most-specific (card) first.
  if (!args.pointerCoordinates) {
    const candidates = rectIntersection({ ...args, droppableContainers: others });
    const cardMatches = matching(candidates, "card");
    if (cardMatches.length > 0) return cardMatches;
    const columnMatches = matching(candidates, "column");
    return columnMatches.length > 0 ? columnMatches : candidates;
  }

  // Columns and lanes first, then only the hovered column's cards: each rect read re-walks the scroll ancestors, and cards outnumber columns.
  const outer = pointerWithin({ ...args, droppableContainers: others.filter((container) => !isCard(container)) });
  const columnMatches = matching(outer, "column");
  const hovered = columnMatches[0];
  const column = hovered && dataOf(hovered);
  if (column?.type !== "column") return outer;
  const cardsInColumn = others.filter((container) => {
    const data = container.data.current as DropTargetData | undefined;
    return data?.type === "card" && data.statusId === column.statusId && data.categoryId === column.categoryId;
  });
  const cardMatches = pointerWithin({ ...args, droppableContainers: cardsInColumn });
  return cardMatches.length > 0 ? cardMatches : columnMatches;
};
