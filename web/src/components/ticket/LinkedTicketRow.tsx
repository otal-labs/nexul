import { Ban, CheckCircle2, Circle, X } from "lucide-react";
import { Link } from "react-router";

import { cn } from "@/lib/utils";
import { linkedTicketKey, type LinkedTicket } from "@/models/TicketLink";

interface LinkedTicketRowProps {
  ticket: LinkedTicket;
  /** A blocker the ticket still waits on shows the blocked icon instead of a neutral circle. */
  waiting?: boolean;
  onRemove?: () => void;
}

const stateOf = (ticket: LinkedTicket, waiting: boolean) => {
  if (ticket.done) return { Icon: CheckCircle2, color: "text-success", label: "Done" };
  if (waiting) return { Icon: Ban, color: "text-warning", label: "Not done yet" };
  return { Icon: Circle, color: "text-muted-foreground", label: "Not done" };
};

export const LinkedTicketRow = ({ ticket, waiting = false, onRemove }: LinkedTicketRowProps) => {
  const { Icon, color, label } = stateOf(ticket, waiting);
  const key = linkedTicketKey(ticket);

  return (
    <li className="flex items-center gap-2 rounded-md px-2 py-1 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <Icon className={cn("size-3.5 shrink-0", color)} role="img" aria-label={label} />
      <Link to={`/tickets/${key}`} className="flex min-w-0 flex-1 items-center gap-2 py-1 text-sm">
        <span className="shrink-0 font-mono text-xs text-muted-foreground">{key}</span>
        <span className="truncate">{ticket.title}</span>
      </Link>
      {onRemove && (
        <button
          type="button"
          aria-label={`Remove link to ${key}`}
          onClick={onRemove}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
        >
          <X className="size-3.5" aria-hidden />
        </button>
      )}
    </li>
  );
};
