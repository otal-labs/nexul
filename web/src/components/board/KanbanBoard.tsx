import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
  type DropAnimation,
} from "@dnd-kit/core";
import { arrayMove } from "@dnd-kit/sortable";
import { useMemo, useState } from "react";

import { SwimlaneSection } from "@/components/board/SwimlaneSection";
import { boardCollisionDetection } from "@/components/board/dragCollisionDetection";
import { boardKeyboardCoordinates } from "@/components/board/dragKeyboardCoordinates";
import { resolveDragMove, ticketsInCell, type DragMoveAction, type DropTargetData } from "@/components/board/dragMove";
import { applyDragPreview, applyDropResult, resolveDragCell, sameDragCell, type DragCell } from "@/components/board/dragPreview";
import { TicketCardOverlay } from "@/components/board/TicketCardOverlay";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface KanbanBoardProps {
  columns: BoardStatus[];
  swimlanes: Swimlane[];
  onDrop: (actions: DragMoveAction[]) => Promise<void> | void;
  onReorderColumns: (statusIds: string[]) => Promise<void> | void;
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

// Same as dnd-kit's default drop animation, plus levelling the tilted overlay card as it glides home.
const dropAnimation: DropAnimation = {
  sideEffects: ({ active, dragOverlay }) => {
    const card = dragOverlay.node.firstElementChild as HTMLElement | null;
    if (card) card.style.rotate = "0deg";
    active.node.style.opacity = "0";
    return () => {
      active.node.style.opacity = "";
    };
  },
};

export interface Swimlane {
  key: string;
  label: string;
  categoryId: string | null;
  tickets: Ticket[];
}

export const KanbanBoard = ({
  columns,
  swimlanes,
  onDrop,
  onReorderColumns,
  onAddTicket,
}: KanbanBoardProps) => {
  // Mouse + Touch (not Pointer) because PointerSensor's pointerdown always wins the race
  // against TouchSensor's touchstart, killing the touch delay and blocking native scroll on small swipes.
  const sensors = useSensors(
    // 2px, not 0: a still click must reach the card's onClick, but any real movement should lift the card at once.
    useSensor(MouseSensor, { activationConstraint: { distance: 2 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: boardKeyboardCoordinates }),
  );

  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [overCell, setOverCell] = useState<DragCell | null>(null);
  // Post-drop order, set inside dnd-kit's drag-end batch so the overlay measures the card in its final slot, held until the drop's refetch lands.
  const [settled, setSettled] = useState<Swimlane[] | null>(null);
  const [settledColumns, setSettledColumns] = useState<BoardStatus[] | null>(null);

  const allTickets = swimlanes.flatMap((lane) => lane.tickets);
  const findTicket = (ticketId: string) => allTickets.find((t) => t.id === ticketId);

  // Purely cosmetic; the real move resolves from the untouched `swimlanes` prop in handleDragEnd below.
  const displaySwimlanes = useMemo(
    () => (activeTicket ? applyDragPreview(swimlanes, activeTicket.id, overCell) : swimlanes),
    [swimlanes, activeTicket, overCell],
  );

  // A column drag has no ticket, so the overlay stays empty and dnd-kit moves the column in place.
  const handleDragStart = (event: DragStartEvent) => {
    setActiveTicket(findTicket(String(event.active.id)) ?? null);
    setOverCell(null);
    setSettled(null);
    setSettledColumns(null);
  };

  const handleDragOver = (event: DragOverEvent) => {
    const active = findTicket(String(event.active.id));
    if (!active) return;
    const target = event.over?.data.current as DropTargetData | undefined;
    // Entering a cell over a card: land above or below it by where the dragged card's midline sits.
    const dragged = event.active.rect.current.translated;
    const overRect = event.over?.rect;
    const after = !!dragged && !!overRect && dragged.top + dragged.height / 2 > overRect.top + overRect.height / 2;
    const next = resolveDragCell({ status: active.status, categoryId: active.category_id }, target, after);
    // Bail out on a same-cell result (functional updater keeps the same reference) so hovering
    // between cards within one column never re-renders (see dragPreview.ts).
    setOverCell((prev) => (sameDragCell(prev, next) ? prev : next));
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTicket(null);
    setOverCell(null);
    const target = event.over?.data.current as DropTargetData | undefined;

    const activeData = event.active.data.current as DropTargetData | undefined;
    if (activeData?.type === "column") {
      if (target?.type !== "column" || target.statusId === activeData.statusId) return;
      const ids = columns.map((column) => column.id);
      const reordered = arrayMove(ids, ids.indexOf(activeData.statusId), ids.indexOf(target.statusId));
      setSettledColumns([...columns].sort((a, b) => reordered.indexOf(a.id) - reordered.indexOf(b.id)));
      void Promise.resolve(onReorderColumns(reordered)).finally(() => setSettledColumns(null));
      return;
    }

    const active = findTicket(String(event.active.id));
    if (!active) return;
    const cell = resolveDragCell({ status: active.status, categoryId: active.category_id }, target);
    if (!cell) return;

    // The target cell as displayed (ghost included) but with real positions, so only tickets that actually moved get an update.
    const realPosition = new Map(allTickets.map((t) => [t.id, t.position]));
    const cellTickets = ticketsInCell(
      displaySwimlanes
        .flatMap((lane) => lane.tickets)
        .map((t) => ({ id: t.id, status: t.status, categoryId: t.category_id, position: t.position })),
      cell.status,
      cell.categoryId,
    ).map((t) => ({ ...t, position: realPosition.get(t.id) ?? t.position }));

    const actions = resolveDragMove(
      { id: active.id, status: active.status, categoryId: active.category_id, position: active.position },
      target,
      cellTickets,
    );
    if (actions.length === 0) return;
    setSettled(applyDropResult(swimlanes, active.id, actions));
    void Promise.resolve(onDrop(actions)).finally(() => setSettled(null));
  };

  const shownSwimlanes = settled ?? displaySwimlanes;
  const shownColumns = settledColumns ?? columns;

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={boardCollisionDetection}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={() => {
        setActiveTicket(null);
        setOverCell(null);
      }}
    >
      {/* One shared horizontal scroll for every lane so column positions stay aligned across swimlanes. */}
      <div className="space-y-6 overflow-x-auto pb-2">
        {shownSwimlanes.map((lane) => (
          <SwimlaneSection key={lane.key} lane={lane} columns={shownColumns} onAddTicket={onAddTicket} />
        ))}
      </div>
      {/* The drop animation lands on the ghost card, already sitting in its final slot thanks to `settled` above. */}
      <DragOverlay dropAnimation={dropAnimation}>
        {activeTicket && <TicketCardOverlay ticket={activeTicket} />}
      </DragOverlay>
    </DndContext>
  );
};
