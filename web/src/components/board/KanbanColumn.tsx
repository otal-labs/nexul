import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { PlusIcon } from "lucide-react";
import { memo, useMemo, useState } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { StatusMark } from "@/components/board/StatusIcon";
import { TicketCard } from "@/components/board/TicketCard";
import type { DropTargetData } from "@/components/board/dragMove";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface KanbanColumnProps {
  droppableId: string;
  laneLabel: string;
  categoryId: string;
  column: BoardStatus;
  collapsed: boolean;
  tickets: Ticket[];
  onAddTicket: (statusId: string) => void;
}

// A column mounts this many cards and grows by as many on "Show more": every card is a sortable, so a long column mounted all at once stalls the board.
const CARD_WINDOW = 10;

const KanbanColumnImpl = ({
  droppableId,
  laneLabel,
  categoryId,
  column,
  collapsed,
  tickets,
  onAddTicket,
}: KanbanColumnProps) => {
  // An empty column folds to a 40px strip with its name set vertically, so the width goes to the columns with cards.
  const empty = tickets.length === 0;
  const [shownCount, setShownCount] = useState(CARD_WINDOW);
  const shown = tickets.length > shownCount ? tickets.slice(0, shownCount) : tickets;
  const hidden = tickets.length - shown.length;
  const data: DropTargetData = { type: "column", statusId: column.id, categoryId, kind: column.kind };
  // Keyed on the ids, not the array: a new items array re-renders every card in the column.
  const idsKey = shown.map((t) => t.id).join(" ");
  const ticketIds = useMemo(() => (idsKey === "" ? [] : idsKey.split(" ")), [idsKey]);
  // Both a drop target for cards and a sortable in its lane's column row; only the header activates the sort.
  const draggable = useActionState({ permission: "projects:write" }).state === "live";
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: droppableId,
    // A collapsed lane's columns keep their full height under the clip, over the lane below, so they take no drop.
    disabled: { draggable: !draggable || collapsed, droppable: collapsed },
    data,
  });

  return (
    <section
      ref={setNodeRef}
      aria-label={`${column.name} column in ${laneLabel}`}
      style={{
        transform: transform ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0)` : undefined,
        transition,
      }}
      className={cn(
        // No well at rest, so the cards sit on the page panel; every column takes its well while a card is held, to read as a drop target.
        "flex flex-col gap-1.5 rounded-lg bg-transparent p-1.5 transition-colors duration-150 ease-standard group-data-[card-drag]/board:bg-surface-2",
        empty ? "w-10 shrink-0 items-center" : "w-50 flex-1",
        // Columns drag in place (no overlay), so the moving one floats above its siblings.
        isDragging && "relative z-10 shadow-elevated",
      )}
    >
      <h4 className={cn("flex gap-1.5 pt-1", empty ? "flex-col items-center" : "items-center")}>
        <span
          ref={setActivatorNodeRef}
          {...(draggable ? { ...attributes, ...listeners, "aria-label": `Reorder ${column.name}` } : {})}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-1.5 rounded-md",
            empty && "flex-col gap-2",
            draggable && "cursor-grab select-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none active:cursor-grabbing",
          )}
        >
          <span className={cn("flex min-w-0 items-center gap-1.5", empty && "flex-col gap-2")}>
            <StatusMark status={column} className="size-3.5" />
            <span className={cn("text-sm font-medium", empty ? "whitespace-nowrap text-muted-foreground [writing-mode:vertical-rl]" : "truncate")}>{column.name}</span>
          </span>
          <span className="font-mono text-xs tabular-nums text-muted-foreground" aria-label={`${tickets.length} tickets`}>
            {tickets.length}
          </span>
        </span>
        <ActionButton
          permission="tickets:write"
          verb="create tickets"
          variant="ghost"
          size="icon"
          icon={PlusIcon}
          onClick={() => onAddTicket(column.id)}
          aria-label={`New ticket in ${column.name}`}
          className="size-6 text-muted-foreground/60"
        />
      </h4>
      {/* The column grows with its cards and the page scrolls; the 1px top padding is room for a card's hover lift. */}
      <EnterList as="div" className={cn("-mt-px flex min-h-16 flex-1 flex-col gap-1.5 pt-px", empty && "w-full")}>
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
        {empty && (
          // The dashed slot only shows while a card is held: then it reads as somewhere a card can land.
          <EmptyRow className="flex-1 rounded-md border border-dashed border-transparent px-0 py-6 text-center text-xs transition-colors duration-150 ease-standard group-data-[card-drag]/board:border-border">
            <span className="sr-only">No tickets</span>
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
  prev.onAddTicket === next.onAddTicket &&
  prev.tickets.length === next.tickets.length &&
  prev.tickets.every((ticket, i) => ticket === next.tickets[i]);

export const KanbanColumn = memo(KanbanColumnImpl, sameColumn);
