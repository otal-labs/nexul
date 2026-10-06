import { UserIcon } from "lucide-react";
import { useState } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { PersonAvatar } from "@/components/PersonAvatar";
import { PersonPickerList } from "@/components/ticket/PersonPickerList";
import { editableRowClass, rowIconClass, rowLabelClass, rowValueClass } from "@/components/ticket/ticketPropertyRowStyle";
import { usePerson } from "@/hooks/PeopleHooks";
import { useSetTicketPerson } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import { personLabel } from "@/models/Person";
import { TicketRole, type Ticket } from "@/models/Ticket";

interface TicketPersonRowProps {
  ticket: Ticket;
  role: TicketRole;
}

const roleLabel = (role: TicketRole) => (role === TicketRole.Tester ? "Tester" : "Developer");

export const TicketPersonRow = ({ ticket, role }: TicketPersonRowProps) => {
  const [open, setOpen] = useState(false);
  const setPerson = useSetTicketPerson();
  const login = role === TicketRole.Tester ? ticket.tester : ticket.developer;
  const person = usePerson(login);
  const name = login ? personLabel(person) : "";
  const label = roleLabel(role);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger className={editableRowClass} aria-label={`${label}: ${name || "no one"}`}>
        <span className={rowIconClass}>
          {login && <PersonAvatar login={login} src={person.avatar_url} className="size-4 text-[8px]" />}
          {!login && <UserIcon className="size-3.5" aria-hidden />}
        </span>
        <span className={rowLabelClass}>{label}</span>
        <span className={cn(rowValueClass, !login && "text-muted-foreground")}>
          {name || "No one"}
        </span>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-56 p-1.5">
        <PersonPickerList
          projectId={ticket.project_id}
          onSelect={(next) => {
            setOpen(false);
            setPerson.mutate({ id: ticket.id, role, login: next });
          }}
        />
      </PopoverContent>
    </Popover>
  );
};
