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
  type Translate,
} from "@dnd-kit/core";
import { arrayMove } from "@dnd-kit/sortable";
import { useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { createPortal } from "react-dom";

import { BoardCardProvider } from "@/components/board/BoardCardProvider";
import { BoardStageHeader } from "@/components/board/BoardStageHeader";
import { boardGridMinWidth, countByStatus } from "@/components/board/boardGrid";
import { useBoardStore } from "@/stores/boardStore";
import { SwimlaneSection } from "@/components/board/SwimlaneSection";
import { boardCollisionDetection } from "@/components/board/dragCollisionDetection";
import { boardKeyboardCoordinates } from "@/components/board/dragKeyboardCoordinates";
import { resolveDragMove, ticketsInCell, type DragMoveAction, type DropTargetData } from "@/components/board/dragMove";
import { applyDragPreview, applyDropResult, resolveDragCell, sameDragCell, type DragCell } from "@/components/board/dragPreview";
import { TicketCardOverlay } from "@/components/board/TicketCardOverlay";
import { dropAnimationFor, landsInDone, startLanding } from "@/components/board/dropMotion";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface KanbanBoardProps {
  projectId: string | undefined;
  /** The chrome's width: the board shares that box and centres inside it, so Create and the last track end together. */
  maxWidth: string;
  /** The scrollbar's width, so the chrome above can end on the same edge as the last track. */
  onGutter?: (width: number) => void;
  columns: BoardStatus[];
  swimlanes: Swimlane[];
  onDrop: (actions: DragMoveAction[]) => Promise<void> | void;
  onReorderColumns: (statusIds: string[]) => Promise<void> | void;
  onAddTicket: (categoryId: string | null, statusId: string) => void;
  /** Shown under the stage row when no lane has a ticket on show, so the stages still read at 0. */
  empty?: ReactNode;
}

export interface Swimlane {
  key: string;
  label: string;
  categoryId: string | null;
  tickets: Ticket[];
}

// Module constants: an options object new each render rebuilds dnd-kit's context and re-renders every card on the board.
// 2px, not 0: a still click must reach the card's onClick, but any real movement should lift the card at once.
const MOUSE_SENSOR = { activationConstraint: { distance: 2 } };
const TOUCH_SENSOR = { activationConstraint: { delay: 200, tolerance: 8 } };
const KEYBOARD_SENSOR = { coordinateGetter: boardKeyboardCoordinates };

export const KanbanBoard = ({
  projectId,
  maxWidth,
  columns,
  swimlanes,
  onDrop,
  onReorderColumns,
  onAddTicket,
  onGutter,
  empty,
}: KanbanBoardProps) => {
  // Mouse + Touch (not Pointer) because PointerSensor's pointerdown always wins the race
  // against TouchSensor's touchstart, killing the touch delay and blocking native scroll on small swipes.
  const sensors = useSensors(
    useSensor(MouseSensor, MOUSE_SENSOR),
    useSensor(TouchSensor, TOUCH_SENSOR),
    useSensor(KeyboardSensor, KEYBOARD_SENSOR),
  );

  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null);
  const [overCell, setOverCell] = useState<DragCell | null>(null);
  // Post-drop order, set inside dnd-kit's drag-end batch so the overlay measures the card in its final slot, held until the drop's refetch lands.
  const [settled, setSettled] = useState<Swimlane[] | null>(null);
  const [settledColumns, setSettledColumns] = useState<BoardStatus[] | null>(null);
  // An over change with the pointer still is the ghost's own layout shift; following it flips between two cells until React gives up.
  const lastOverDelta = useRef<Translate | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  // A page scrollbar inside the board narrows its tracks; the chrome above is told the same width, so both end on one edge.
  const gutterRef = useRef<number | undefined>(undefined);
  const [overflowing, setOverflowing] = useState(false);
  const [moreRight, setMoreRight] = useState(false);
  const [gutter, setGutter] = useState(0);
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const report = () => {
      const gutter = el.offsetWidth - el.clientWidth;
      setOverflowing(el.scrollWidth > el.clientWidth + 1);
      setMoreRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1);
      if (gutter === gutterRef.current) return;
      gutterRef.current = gutter;
      setGutter(gutter);
      onGutter?.(gutter);
    };
    report();
    const observer = new ResizeObserver(report);
    observer.observe(el);
    return () => observer.disconnect();
  }, [onGutter]);
  const dropAnimation = useMemo(() => dropAnimationFor(), []);

  const allTickets = swimlanes.flatMap((lane) => lane.tickets);
  const findTicket = (ticketId: string) => allTickets.find((t) => t.id === ticketId);

  // Purely cosmetic; the real move resolves from the untouched `swimlanes` prop in handleDragEnd below.
  const displaySwimlanes = useMemo(
    () => (activeTicket ? applyDragPreview(swimlanes, activeTicket.id, overCell) : swimlanes),
    [swimlanes, activeTicket, overCell],
  );

  // A column drag has no ticket, so the overlay stays empty and dnd-kit moves the column in place.
  const handleDragStart = (event: DragStartEvent) => {
    startLanding(event.activatorEvent instanceof KeyboardEvent);
    setActiveTicket(findTicket(String(event.active.id)) ?? null);
    setOverCell(null);
    lastOverDelta.current = null;
    setSettled(null);
    setSettledColumns(null);
  };

  const handleDragOver = (event: DragOverEvent) => {
    const active = findTicket(String(event.active.id));
    if (!active) return;
    const pointerStill = lastOverDelta.current?.x === event.delta.x && lastOverDelta.current?.y === event.delta.y;
    lastOverDelta.current = event.delta;
    if (pointerStill) return;
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
    const kindOf = (statusId: string) => columns.find((column) => column.id === statusId)?.kind;
    landsInDone(kindOf(cell.status) === "done" && kindOf(active.status) !== "done");
    setSettled(applyDropResult(swimlanes, active.id, actions));
    void Promise.resolve(onDrop(actions)).finally(() => setSettled(null));
  };

  const shownSwimlanes = settled ?? displaySwimlanes;
  const shownColumns = settledColumns ?? columns;
  const doneOpen = useBoardStore((s) => s.doneOpen);
  const counts = countByStatus(shownSwimlanes);

  return (
    <BoardCardProvider projectId={projectId}>
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
        {/* The board scrolls on its own, both ways, so the stage header and each lane band can pin while the toolbar stays put. */}
        {/* The frame reaches into the page's right padding (Container's px-6) and the scrollbar lives there, so opening a lane never narrows the columns. */}
        <div
          className="relative mx-auto mt-3 flex min-h-0 w-full flex-1 flex-col"
          style={{ maxWidth, "--board-gutter": `${gutter}px` } as CSSProperties}
        >
          {/* It runs to the panel's bottom and fades there; below 1024 the right edge fades too, so a cut column reads as more board. */}
          <div
            ref={scrollRef}
            onScroll={() => {
              const el = scrollRef.current;
              if (el) setMoreRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1);
            }}
            data-card-drag={activeTicket ? "" : undefined}
            className="group/board @container min-h-0 flex-1 overflow-auto pb-6 [scrollbar-gutter:stable] [mask-image:linear-gradient(to_bottom,#000_calc(100%-1.5rem),transparent)]"
          >
            <div className="relative mx-auto min-h-full w-full" style={{ minWidth: boardGridMinWidth(shownColumns, doneOpen) }}>
              <div className="relative space-y-3">
                <BoardStageHeader columns={shownColumns} counts={counts} onAddTicket={onAddTicket} overflowing={overflowing} />
                {shownSwimlanes.map((lane) => (
                  <SwimlaneSection key={lane.key} lane={lane} columns={shownColumns} overflowing={overflowing} />
                ))}
                {empty}
              </div>
            </div>
          </div>
          {/* Pinned to the scroll box's right edge, so a cut card fades where it is cut rather than at the grid's far end. */}
          {moreRight && (
            <div
              aria-hidden
              // Above the pinned stage row and lane bands, so a cut stage fades like a cut card; the band counts sit outside it.
              className="pointer-events-none absolute inset-y-0 right-[var(--board-gutter,0px)] z-30 w-3 bg-gradient-to-l from-[var(--panel)] to-transparent"
            />
          )}
        </div>
        {/* The drop animation lands on the ghost card, already sitting in its final slot thanks to `settled` above. */}
        {/* Portalled to body: the page frame's backdrop-filter would otherwise anchor the fixed overlay to the frame, not the viewport. */}
        {createPortal(
          <DragOverlay dropAnimation={dropAnimation}>
            {activeTicket && <TicketCardOverlay ticket={activeTicket} />}
          </DragOverlay>,
          document.body,
        )}
      </DndContext>
    </BoardCardProvider>
  );
};
