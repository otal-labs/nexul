import { arrayMove } from "@dnd-kit/sortable";

// A column/card drop carries status and category, a lane drop only category; `kind` limits column reorders to one stage.
export type DropTargetData =
  | { type: "column"; statusId: string; categoryId: string; kind?: string }
  | { type: "lane"; categoryId: string }
  | { type: "card"; ticketId: string; statusId: string; categoryId: string };

export type DragMoveAction =
  | { kind: "status"; ticketId: string; status: string }
  | { kind: "category"; ticketId: string; categoryId: string }
  | { kind: "reorder"; updates: { ticketId: string; position: number }[] };

/** The dragged ticket, and the other tickets it's shared some drop target with. */
export interface DragTicket {
  id: string;
  status: string;
  categoryId: string;
  position: number;
}

// Filters by categoryId too, not just status: a lane can span several real categoryIds (merged by name).
export const ticketsInCell = (tickets: DragTicket[], status: string, categoryId: string): DragTicket[] =>
  tickets.filter((t) => t.status === status && t.categoryId === categoryId).sort((a, b) => a.position - b.position);

// dnd-kit's arrayMove when the active is already previewed in the cell, a plain insert at the target when it isn't.
const placeInCell = (cellTickets: DragTicket[], active: DragTicket, overTicketId: string | undefined): DragTicket[] => {
  const from = cellTickets.findIndex((t) => t.id === active.id);
  const over = cellTickets.findIndex((t) => t.id === overTicketId);
  if (from === -1) {
    const insertAt = over === -1 ? cellTickets.length : over;
    return [...cellTickets.slice(0, insertAt), active, ...cellTickets.slice(insertAt)];
  }
  if (over === -1) return cellTickets;
  return arrayMove(cellTickets, from, over);
};

export const resolveDragMove = (
  active: DragTicket,
  target: DropTargetData | undefined,
  // The target cell's tickets in displayed order (the previewed ghost included when it's there), carrying their real positions.
  cellTickets: DragTicket[],
): DragMoveAction[] => {
  if (!target) return [];
  if (target.type === "card" && target.ticketId === active.id) return [];

  // A lane drop carries only category, so status stays put; a column/card drop carries both.
  const targetStatus = target.type === "lane" ? active.status : target.statusId;
  const targetCategory = target.categoryId;

  const actions: DragMoveAction[] = [];
  if (targetStatus !== active.status) actions.push({ kind: "status", ticketId: active.id, status: targetStatus });
  if (targetCategory !== active.categoryId) actions.push({ kind: "category", ticketId: active.id, categoryId: targetCategory });

  // A status/category change makes the backend append the ticket to its new cell, so its slot must always be sent explicitly.
  const cellChanged = actions.length > 0;
  const ordered = placeInCell(cellTickets, active, target.type === "card" ? target.ticketId : undefined);
  const updates = ordered
    .map((ticket, position) => ({ ticketId: ticket.id, position }))
    .filter((update, index) => ordered[index]?.position !== update.position || (cellChanged && update.ticketId === active.id));
  if (updates.length > 0) actions.push({ kind: "reorder", updates });
  return actions;
};
