import type { TicketStatus } from "@/models/Ticket";

// The ticket at the other end of a found-in or blocked-by link; done reads its column's stage, never the name.
export interface LinkedTicket {
  id: string;
  project_id: string;
  prefix: string;
  number: number;
  title: string;
  status: TicketStatus;
  done: boolean;
}

// Both directions of a ticket's links; blocked holds while any blocker sits outside a done-stage column.
export interface TicketLinkSet {
  found_in: LinkedTicket | null;
  origin_unknown: boolean;
  bugs_found: LinkedTicket[];
  blocked_by: LinkedTicket[];
  blocks: LinkedTicket[];
  blocked: boolean;
}

// Blocked ticket id to the blockers it still waits on; a ticket with none is absent.
export type BlockersByTicket = Record<string, LinkedTicket[]>;

export const linkedTicketKey = (ticket: LinkedTicket): string =>
  ticket.prefix ? `${ticket.prefix}-${ticket.number}` : ticket.id;
