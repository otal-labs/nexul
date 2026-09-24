import { Bot } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import { ReporterKind, reporterName, reporterOnBehalfOf, type Ticket } from "@/models/Ticket";

interface TicketReporterRowProps {
  ticket: Ticket;
}

// Read-only: the reporter is set once when the ticket is filed.
export const TicketReporterRow = ({ ticket }: TicketReporterRowProps) => {
  const { reporter } = ticket;
  const isNexul = reporter.kind !== ReporterKind.User;
  const name = reporterName(reporter);
  const onBehalfOf = reporterOnBehalfOf(reporter);

  return (
    <div className={rowClass}>
      {isNexul && (
        <span className="flex size-4 shrink-0 items-center justify-center rounded-full bg-accent text-accent-foreground">
          <Bot className="size-3" aria-hidden />
        </span>
      )}
      {!isNexul && name && <PersonAvatar login={name} className="size-4 text-[8px]" />}
      <span className="w-16 shrink-0 text-xs text-muted-foreground">Reporter</span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate font-mono text-xs text-foreground">{name || "Unknown"}</span>
        {onBehalfOf && <span className="truncate text-[11px] text-muted-foreground">{onBehalfOf}</span>}
      </span>
    </div>
  );
};
