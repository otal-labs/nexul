import type { DragMoveAction, DropTargetData } from "@/components/board/dragMove";
import type { Swimlane } from "@/components/board/KanbanBoard";
import type { Ticket, TicketStatus } from "@/models/Ticket";

export interface DragCell {
  status: string;
  categoryId: string;
  // Read once, on entering the cell: which card the ghost lands beside. Inside a cell dnd-kit's SortableContext previews on its own.
  anchor?: { ticketId: string; after: boolean };
}

export const resolveDragCell = (
  active: { status: string; categoryId: string },
  target: DropTargetData | undefined,
  after = false,
): DragCell | null => {
  if (!target) return null;
  if (target.type === "lane") return { status: active.status, categoryId: target.categoryId };
  if (target.type === "column") return { status: target.statusId, categoryId: target.categoryId };
  return { status: target.statusId, categoryId: target.categoryId, anchor: { ticketId: target.ticketId, after } };
};

// Cell identity only: a different anchor inside the same cell must not re-render, or the ghost would jump under the pointer.
export const sameDragCell = (a: DragCell | null, b: DragCell | null): boolean =>
  a === b || (a?.status === b?.status && a?.categoryId === b?.categoryId);

// Ghost-moves the ticket into its target cell so siblings make room; purely cosmetic, the real move resolves on drop.
export const applyDragPreview = (swimlanes: Swimlane[], activeTicketId: string, cell: DragCell | null): Swimlane[] => {
  if (!cell) return swimlanes;

  const active = swimlanes.flatMap((lane) => lane.tickets).find((t) => t.id === activeTicketId);
  if (!active) return swimlanes;
  // Already in this cell — dnd-kit's own SortableContext handles the preview.
  if (cell.status === active.status && cell.categoryId === active.category_id) return swimlanes;

  return swimlanes.map((lane) => {
    const withoutActive = lane.tickets.filter((t) => t.id !== activeTicketId);
    if ((lane.categoryId ?? "") !== cell.categoryId) return { ...lane, tickets: withoutActive };

    const cellTickets = withoutActive.filter((t) => t.status === cell.status).sort((a, b) => a.position - b.position);
    const anchorIndex = cellTickets.findIndex((t) => t.id === cell.anchor?.ticketId);
    const insertAt = anchorIndex === -1 ? cellTickets.length : anchorIndex + (cell.anchor?.after ? 1 : 0);
    const ghost: Ticket = { ...active, status: cell.status as TicketStatus, category_id: cell.categoryId };
    cellTickets.splice(insertAt, 0, ghost);
    // Renumbered so the column's sort-by-position shows the ghost exactly where it slid in.
    const renumbered = cellTickets.map((t, position) => ({ ...t, position }));
    return { ...lane, tickets: [...withoutActive.filter((t) => t.status !== cell.status), ...renumbered] };
  });
};

// The board as it will read once a drop's actions have landed, so the dropped card sits in its final slot before the server answers.
export const applyDropResult = (swimlanes: Swimlane[], activeTicketId: string, actions: DragMoveAction[]): Swimlane[] => {
  const active = swimlanes.flatMap((lane) => lane.tickets).find((t) => t.id === activeTicketId);
  if (!active || actions.length === 0) return swimlanes;

  const status = actions.find((a) => a.kind === "status")?.status ?? active.status;
  const categoryId = actions.find((a) => a.kind === "category")?.categoryId ?? active.category_id;
  const positions = new Map(
    actions.flatMap((a) => (a.kind === "reorder" ? a.updates.map((u) => [u.ticketId, u.position] as const) : [])),
  );
  const moved: Ticket = { ...active, status: status as TicketStatus, category_id: categoryId };

  return swimlanes.map((lane) => {
    const others = lane.tickets.filter((t) => t.id !== activeTicketId);
    const tickets = (lane.categoryId ?? "") === categoryId ? [...others, moved] : others;
    return { ...lane, tickets: tickets.map((t) => (positions.has(t.id) ? { ...t, position: positions.get(t.id)! } : t)) };
  });
};
