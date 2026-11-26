import { namesAccount } from "@nexul/client-core/ticketPeople";

import { resolveStatus } from "@/hooks/StatusHooks";
import { StatusKind, type BoardStatus } from "@/models/Status";
import { cardPerson, type Ticket } from "@/models/Ticket";

// Work in motion first: what is being built, then reviewed, then tested, then what waits in the backlog.
const STAGE_ORDER: StatusKind[] = [StatusKind.Progress, StatusKind.Review, StatusKind.Testing, StatusKind.Backlog];

export interface TicketOnYou {
  ticket: Ticket;
  status: Pick<BoardStatus, "name" | "kind" | "icon"> | undefined;
}

// The open tickets whose next move is the account's, by the board card's rule (the tester while testing, else the developer).
export const ticketsOnYou = (tickets: Ticket[], statuses: BoardStatus[] | undefined, account: { id: string; login: string }): TicketOnYou[] =>
  tickets
    .map((ticket) => ({ ticket, status: resolveStatus(statuses, ticket.status) }))
    .filter(({ ticket, status }) => status?.kind !== StatusKind.Done && namesAccount(cardPerson(ticket, status?.kind), account))
    .sort((a, b) => stageRank(a.status?.kind) - stageRank(b.status?.kind) || b.ticket.updated_at.localeCompare(a.ticket.updated_at));

const stageRank = (kind: StatusKind | undefined) => {
  const rank = kind ? STAGE_ORDER.indexOf(kind) : -1;
  return rank === -1 ? STAGE_ORDER.length : rank;
};
