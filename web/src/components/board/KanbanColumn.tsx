import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { memo, useMemo, useState } from "react";

import { TicketCard } from "@/components/board/TicketCard";
import type { DropTargetData } from "@/components/board/dragMove";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface KanbanColumnProps {
  droppableId: string;
  laneLabel: string;
  categoryId: string;
  column: BoardStatus;
  collapsed: boolean;
  // Done folded to its strip: the cell still takes drops, its cards are counted in the stage header instead.
  folded: boolean;
  tickets: Ticket[];
}

// A column mounts this many cards and grows by as many on "Show more": every card is a sortable, so a long column mounted all at once stalls the board.
const CARD_WINDOW = 10;

// One stage's cell in one lane: its own cards on the stage's well, as tall as the lane so a drop below its last card lands in it; the stage header is drawn once above every lane.
// Folded Done has no well: its cell is bare until a card is held, when it shows where the card would land.
const KanbanColumnImpl = ({ droppableId, laneLabel, categoryId, column, collapsed, folded, tickets }: KanbanColumnProps) => {
  const empty = tickets.length === 0;
  const [shownCount, setShownCount] = useState(CARD_WINDOW);
  const windowed = tickets.length > shownCount ? tickets.slice(0, shownCount) : tickets;
  const shown = folded ? [] : windowed;
  const hidden = folded ? 0 : tickets.length - shown.length;
  const data: DropTargetData = { type: "column", statusId: column.id, categoryId, kind: column.kind };
  // Keyed on the ids, not the array: a new items array re-renders every card in the column.
  const idsKey = shown.map((t) => t.id).join(" ");
  const ticketIds = useMemo(() => (idsKey === "" ? [] : idsKey.split(" ")), [idsKey]);
  // A collapsed lane's cells keep their height under the clip, over the lane below, so they take no drop.
  const { setNodeRef } = useDroppable({ id: droppableId, disabled: collapsed, data });

  return (
    <section
      ref={setNodeRef}
      aria-label={`${column.name} column in ${laneLabel}`}
      className={cn(
        "flex min-w-0 flex-col gap-1.5 rounded-lg p-1.5",
        // An empty cell is a place to land, not a slab: half the well, same footprint, still a drop target.
        empty && "bg-well-shade/50",
        !empty && (folded ? "group-data-[card-drag]/board:bg-well-shade" : "bg-well-shade"),
      )}
    >
      {/* The 1px top padding is room for a card's hover lift. */}
      <EnterList as="div" className="-mt-px flex flex-col gap-1.5 pt-px">
        <SortableContext items={ticketIds} strategy={verticalListSortingStrategy}>
          {shown.map((ticket) => (
            <TicketCard key={ticket.id} ticket={ticket} collapsed={collapsed} />
          ))}
        </SortableContext>
        {hidden > 0 && (
          <button
            type="button"
            onClick={() => setShownCount((count) => count + CARD_WINDOW)}
            className="shrink-0 rounded-md px-2 py-1.5 text-left text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground"
          >
            Show {Math.min(hidden, CARD_WINDOW)} more of <span className="font-mono tabular-nums">{hidden}</span>
          </button>
        )}
        {(empty || folded) && (
          // Card-tall while a card is held, so an empty cell or the folded Done strip reads as somewhere to land.
          <EmptyRow
            flush
            className={cn(
              "h-6 rounded-md border border-dashed border-transparent transition-colors duration-150 ease-standard group-data-[card-drag]/board:h-20 group-data-[card-drag]/board:border-border",
            )}
          >
            <span className="sr-only">{folded ? `${tickets.length} tickets, folded` : "No tickets"}</span>
          </EmptyRow>
        )}
      </EnterList>
    </section>
  );
};

// The lane rebuilds every column's tickets on each refetch; equal tickets in the same order mean nothing here changed.
const sameColumn = (prev: KanbanColumnProps, next: KanbanColumnProps) =>
  prev.droppableId === next.droppableId &&
  prev.laneLabel === next.laneLabel &&
  prev.categoryId === next.categoryId &&
  prev.column === next.column &&
  prev.collapsed === next.collapsed &&
  prev.folded === next.folded &&
  prev.tickets.length === next.tickets.length &&
  prev.tickets.every((ticket, i) => ticket === next.tickets[i]);

export const KanbanColumn = memo(KanbanColumnImpl, sameColumn);
