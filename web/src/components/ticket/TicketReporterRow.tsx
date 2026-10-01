import { Bot } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import { usePerson } from "@/hooks/PeopleHooks";
import { personLabel } from "@/models/Person";
import { ReporterKind, reporterName, reporterOnBehalfOf, type Ticket } from "@/models/Ticket";

interface TicketReporterRowProps {
  ticket: Ticket;
}

// Read-only: the reporter is set once when the ticket is filed.
export const TicketReporterRow = ({ ticket }: TicketReporterRowProps) => {
  const { reporter } = ticket;
  const isNexul = reporter.kind !== ReporterKind.User;
  const person = usePerson(reporter.login ?? "");
  const nameOf = () => personLabel(person);
  const name = reporterName(reporter, nameOf);
  const onBehalfOf = reporterOnBehalfOf(reporter, nameOf);

  return (
    <div className={rowClass}>
      {isNexul && (
        <span className="flex size-4 shrink-0 items-center justify-center rounded-full bg-accent text-accent-foreground">
          <Bot className="size-3" aria-hidden />
        </span>
      )}
      {!isNexul && name && <PersonAvatar login={person.login} src={person.avatar_url} className="size-4 text-[8px]" />}
      <span className="w-16 shrink-0 text-xs text-muted-foreground">Reporter</span>
      <span className="min-w-0 truncate text-xs">
        <span className="text-foreground">{name || "Unknown"}</span>
        {onBehalfOf && <span className="text-muted-foreground"> {onBehalfOf}</span>}
      </span>
    </div>
  );
};
