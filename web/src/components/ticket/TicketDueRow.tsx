import { CalendarIcon, XIcon } from "lucide-react";
import { useState, type KeyboardEvent } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { PropertyRowTrigger } from "@/components/ticket/PropertyRowTrigger";
import { rowIconClass, rowLabelClass, rowValueClass } from "@/components/ticket/ticketPropertyRowStyle";
import { useSetTicketDueOn } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/models/Ticket";
import { formatDay } from "@/utils/TimeUtility";

interface TicketDueRowProps {
  ticket: Ticket;
}

// The date saves when the popover closes, not on every keystroke: typing a year passes through 0002, 0020 and 0202.
export const TicketDueRow = ({ ticket }: TicketDueRowProps) => {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(ticket.due_on);
  const setDueOn = useSetTicketDueOn();
  const text = ticket.due_on ? formatDay(ticket.due_on) : "No due date";

  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) setDraft(ticket.due_on);
    if (!next && draft !== ticket.due_on) setDueOn.mutate({ id: ticket.id, dueOn: draft });
  };

  // A half-typed date reads as empty; clearing has its own button, so a gap never saves.
  const pick = (value: string) => {
    if (value !== "") setDraft(value);
  };

  const saveOnEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") onOpenChange(false);
  };

  const clear = () => {
    setOpen(false);
    setDueOn.mutate({ id: ticket.id, dueOn: "" });
  };

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <PropertyRowTrigger projectId={ticket.project_id} verb="set a ticket's due date" label={`Due: ${ticket.due_on ? text : "none"}`}>
          <span className={rowIconClass}>
            <CalendarIcon className="size-3.5" aria-hidden />
          </span>
          <span className={rowLabelClass}>Due</span>
          <span className={cn(rowValueClass, !ticket.due_on && "text-muted-foreground")}>{text}</span>
        </PropertyRowTrigger>
      </PopoverTrigger>
      <PopoverContent align="start" className="flex w-56 items-center gap-1.5 p-1.5">
        <Input
          type="date"
          aria-label="Due date"
          value={draft}
          onChange={(event) => pick(event.target.value)}
          onKeyDown={saveOnEnter}
          className="h-8 min-w-0 flex-1 text-xs"
        />
        {ticket.due_on !== "" && (
          <ActionButton
            permission="tickets:write"
            projectId={ticket.project_id}
            verb="set a ticket's due date"
            variant="ghost"
            size="icon"
            icon={XIcon}
            aria-label="Clear due date"
            onClick={clear}
            className="size-8 shrink-0 text-muted-foreground"
          />
        )}
      </PopoverContent>
    </Popover>
  );
};
