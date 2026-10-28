import { personLabel } from "@nexul/client-core/person";

import { formatUpdatedAgo } from "@/components/doc/docTime";
import { usePerson } from "@/hooks/PeopleHooks";
import { reporterLabel, type Ticket } from "@/models/Ticket";

interface TicketCreatedLineProps {
  ticket: Ticket;
}

// The rail's foot: when the ticket was filed and by whom, including an agent or automation reporter, read only.
export const TicketCreatedLine = ({ ticket }: TicketCreatedLineProps) => {
  const person = usePerson(ticket.reporter.login ?? "");
  const reporter = reporterLabel(ticket.reporter, () => personLabel(person));
  return (
    <p className="border-t border-border px-2 pt-4 font-mono text-xs text-muted-foreground">
      created {formatUpdatedAgo(ticket.created_at)}
      {reporter && ` by ${reporter}`}
    </p>
  );
};
