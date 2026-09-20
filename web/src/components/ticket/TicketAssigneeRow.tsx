import { UserIcon } from "lucide-react";
import { useState } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ReviewerAvatars } from "@/components/codereview/ReviewerAvatars";
import { AssigneePickerList } from "@/components/ticket/AssigneePickerList";
import { editableRowClass, rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import type { Ticket } from "@/models/Ticket";

interface TicketAssigneeRowProps {
  ticket: Ticket;
  onSetAssignee?: (ticketId: string, assignee: string) => Promise<void> | void;
}

export const TicketAssigneeRow = ({ ticket, onSetAssignee }: TicketAssigneeRowProps) => {
  const [open, setOpen] = useState(false);
  const content = ticket.assignee ? (
    <>
      <ReviewerAvatars names={[ticket.assignee]} />
      <span className="font-mono text-xs text-foreground">{ticket.assignee}</span>
    </>
  ) : (
    <>
      <UserIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <span className="font-mono text-xs text-muted-foreground">Unassigned</span>
    </>
  );

  if (!onSetAssignee) {
    return (
      <div className={rowClass}>
        <span className="sr-only">Assignee</span>
        {content}
      </div>
    );
  }

  return (
    <>
      <span className="sr-only">Assignee</span>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger className={editableRowClass}>{content}</PopoverTrigger>
        <PopoverContent align="start" className="w-56 p-1.5">
          <AssigneePickerList
            onSelect={(login) => {
              setOpen(false);
              void onSetAssignee(ticket.id, login);
            }}
          />
        </PopoverContent>
      </Popover>
    </>
  );
};
