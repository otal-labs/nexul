import { SquareKanban } from "lucide-react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { StatusMark } from "@/components/board/StatusIcon";
import { StatusTransitionButtons } from "@/components/ticket/StatusTransitionButtons";
import {
  editableRowClass,
  rowClass,
  rowIconClass,
  rowLabelClass,
  rowValueClass,
} from "@/components/ticket/ticketPropertyRowStyle";
import { useTicketStatus } from "@/hooks/StatusHooks";
import type { Ticket } from "@/models/Ticket";

interface TicketStatusRowProps {
  ticket: Ticket;
  onTransition?: (statusId: string) => Promise<void> | void;
}

export const TicketStatusRow = ({ ticket, onTransition }: TicketStatusRowProps) => {
  const { status, isPending } = useTicketStatus(ticket);
  const name = status?.name ?? (isPending ? "" : "Unknown status");
  const content = (
    <>
      <span className={rowIconClass}>
        <SquareKanban className="size-3.5" aria-hidden />
      </span>
      <span className={rowLabelClass}>Status</span>
      <span className="flex min-w-0 items-center gap-1.5">
        {status && <StatusMark status={status} className="size-3.5" />}
        <span className={rowValueClass}>{name}</span>
      </span>
    </>
  );

  if (!onTransition) return <div className={rowClass}>{content}</div>;

  return (
    <Popover>
      <PopoverTrigger className={editableRowClass} aria-label={`Status: ${name}`}>
        {content}
      </PopoverTrigger>
      <PopoverContent className="w-44 p-1" align="start">
        <StatusTransitionButtons ticket={ticket} onTransition={onTransition} />
      </PopoverContent>
    </Popover>
  );
};
