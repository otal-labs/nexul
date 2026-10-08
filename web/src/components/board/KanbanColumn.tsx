import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { GripVerticalIcon, PlusIcon } from "lucide-react";

import { StatusMark } from "@/components/board/StatusIcon";
import { TicketCard } from "@/components/board/TicketCard";
import type { DropTargetData } from "@/components/board/dragMove";
import { EmptyRow } from "@/components/EmptyRow";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface KanbanColumnProps {
  droppableId: string;
  laneLabel: string;
  categoryId: string;
  column: BoardStatus;
  tickets: Ticket[];
  onAddTicket: (statusId: string) => void;
}

export const KanbanColumn = ({
  droppableId,
  laneLabel,
  categoryId,
  column,
  tickets,
  onAddTicket,
}: KanbanColumnProps) => {
  const data: DropTargetData = { type: "column", statusId: column.id, categoryId, kind: column.kind };
  // Both a drop target for cards and a sortable in its lane's column row; only the grip activates the sort.
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: droppableId,
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
        "flex w-72 shrink-0 flex-col gap-1.5 rounded-lg bg-muted p-1.5",
        // Columns drag in place (no overlay), so the moving one floats above its siblings.
        isDragging && "relative z-10 shadow-elevated",
      )}
    >
      <h4 className="flex items-center gap-1.5 px-1 pt-1">
        <button
          ref={setActivatorNodeRef}
          type="button"
          aria-label={`Reorder ${column.name}`}
          className="cursor-grab rounded-md p-0.5 text-muted-foreground/60 hover:text-foreground active:cursor-grabbing focus-visible:ring-[3px] focus-visible:ring-ring/50"
          {...attributes}
          {...listeners}
        >
          <GripVerticalIcon className="size-3.5" aria-hidden />
        </button>
        <span className="flex min-w-0 items-center gap-1.5">
          <StatusMark status={column} className="size-3.5" />
          <span className="truncate text-sm font-medium">{column.name}</span>
        </span>
        <span className="font-mono text-xs tabular-nums text-muted-foreground" aria-label={`${tickets.length} tickets`}>
          {tickets.length}
        </span>
        <span className="flex-1" />
        <button
          type="button"
          onClick={() => onAddTicket(column.id)}
          aria-label={`New ticket in ${column.name}`}
          className="rounded-md p-1 text-muted-foreground/60 hover:bg-accent hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
        >
          <PlusIcon className="size-4" aria-hidden />
        </button>
      </h4>
      {/* Capped at five two-line cards plus gaps (31.5rem) so scrolling depends on the ticket count, not the window height. */}
      <div className="flex max-h-[31.5rem] min-h-16 flex-col gap-1.5 overflow-y-auto">
        <SortableContext items={tickets.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {tickets.map((ticket, index) => (
            <TicketCard key={ticket.id} ticket={ticket} index={index} />
          ))}
        </SortableContext>
        {tickets.length === 0 && (
          <EmptyRow className="px-2 py-3">No tickets</EmptyRow>
        )}
      </div>
    </section>
  );
};
