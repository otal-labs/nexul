import type { ActiveTargets } from "@/hooks/TrailHooks";
import type { Ticket } from "@/models/Ticket";

export interface TicketRun {
  ticket: Ticket;
  waiting: boolean;
  started: string;
}

// Runs waiting on an answer lead, since they stand still until someone gives one; the rest newest first.
export const ticketRuns = (targets: ActiveTargets, tickets: Ticket[]): TicketRun[] => {
  const byId = new Map(tickets.map((t) => [t.id, t]));
  return Object.keys(targets.active)
    .flatMap((id) => {
      const ticket = byId.get(id);
      return ticket ? [{ ticket, waiting: id in targets.waiting, started: targets.started[id] ?? "" }] : [];
    })
    .sort((a, b) => Number(b.waiting) - Number(a.waiting) || b.started.localeCompare(a.started));
};
