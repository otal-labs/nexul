import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { StatusTransitionButtons } from "@/components/ticket/StatusTransitionButtons";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import { editableRowClass, rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import type { Ticket, TicketStatus as TicketStatusType } from "@/models/Ticket";

interface TicketStatusRowProps {
  ticket: Ticket;
  onTransition?: (status: TicketStatusType) => Promise<void> | void;
}

export const TicketStatusRow = ({ ticket, onTransition }: TicketStatusRowProps) => {
  if (!onTransition) {
    return (
      <div className={rowClass}>
        <span className="sr-only">Status</span>
        <TicketStatusBadge status={ticket.status} />
      </div>
    );
  }
  return (
    <>
      <span className="sr-only">Status</span>
      <Popover>
        <PopoverTrigger className={editableRowClass}>
          <TicketStatusBadge status={ticket.status} />
        </PopoverTrigger>
        <PopoverContent className="w-44 p-1" align="start">
          <StatusTransitionButtons ticket={ticket} onTransition={onTransition} />
        </PopoverContent>
      </Popover>
    </>
  );
};
