import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { PlusIcon } from "lucide-react";
import { memo, useMemo } from "react";

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

const KanbanColumnImpl = ({
  droppableId,
  laneLabel,
  categoryId,
  column,
  collapsed,
  tickets,
  onAddTicket,
}: KanbanColumnProps) => {
  const data: DropTargetData = { type: "column", statusId: column.id, categoryId, kind: column.kind };
  // Keyed on the ids, not the array: a new items array re-renders every card in the column.
  const idsKey = tickets.map((t) => t.id).join(" ");
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
        "flex w-72 shrink-0 flex-col gap-1.5 rounded-lg bg-surface-2 p-1.5",
        // Columns drag in place (no overlay), so the moving one floats above its siblings.
        isDragging && "relative z-10 shadow-elevated",
      )}
    >
      <h4 className="flex items-center gap-1.5 px-1 pt-1">
        <span
          ref={setActivatorNodeRef}
          {...(draggable ? { ...attributes, ...listeners, "aria-label": `Reorder ${column.name}` } : {})}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-1.5 rounded-md",
            draggable && "cursor-grab select-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none active:cursor-grabbing",
          )}
        >
          <span className="flex min-w-0 items-center gap-1.5">
            <StatusMark status={column} className="size-3.5" />
            <span className="truncate text-sm font-medium">{column.name}</span>
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
      {/* Capped at five two-line cards plus gaps (31.5rem) so scrolling depends on the ticket count; the 1px top padding is room for a card's hover lift. */}
      <EnterList as="div" className="-mt-px flex max-h-[calc(31.5rem+1px)] min-h-16 flex-col gap-1.5 overflow-y-auto pt-px">
        <SortableContext items={ticketIds} strategy={verticalListSortingStrategy}>
          {tickets.map((ticket) => (
            <TicketCard key={ticket.id} ticket={ticket} collapsed={collapsed} />
          ))}
        </SortableContext>
        {tickets.length === 0 && (
          // A dashed slot reads as somewhere a card can land.
          <EmptyRow className="rounded-md border border-dashed border-border px-2 py-6 text-center text-xs">No tickets</EmptyRow>
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
