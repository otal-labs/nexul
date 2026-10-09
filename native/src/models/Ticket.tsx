import { StatusKind } from "@/models/Status";

export const TicketRole = {
  Developer: "developer",
  Tester: "tester",
} as const;

export type TicketRole = (typeof TicketRole)[keyof typeof TicketRole];

export interface Ticket {
  id: string;
  project_id: string;
  type_id: string;
  title: string;
  body: string;
  // The board column id (a BoardStatus.id), not a fixed lifecycle value — matches what the web board groups on.
  status: string;
  position: number;
  // Per-project sequential number, combined with the project's prefix for a human-readable PREFIX-N id.
  number: number;
  // Member logins; empty when nobody holds the role.
  developer: string;
  tester: string;
  created_at: string;
  updated_at: string;
  // The gateway serializes labels as null when a ticket has none.
  labels: string[] | null;
}

// Links use the human-readable PREFIX-NUMBER key, falling back to the raw id when the prefix isn't at hand.
export const ticketKey = (ticket: Ticket, prefix: string | undefined): string =>
  prefix ? `${prefix}-${ticket.number}` : ticket.id;

// A board card shows who acts next: the tester while the ticket sits in a testing-stage column, else the developer.
export const cardPerson = (ticket: Ticket, stage: StatusKind | undefined): { role: TicketRole; login: string } =>
  stage === StatusKind.Testing
    ? { role: TicketRole.Tester, login: ticket.tester }
    : { role: TicketRole.Developer, login: ticket.developer };
