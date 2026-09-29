import { UserIcon } from "lucide-react";
import { useState } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { PersonAvatar } from "@/components/PersonAvatar";
import { PersonPickerList } from "@/components/ticket/PersonPickerList";
import { editableRowClass } from "@/components/ticket/ticketPropertyRowStyle";
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
        {login && <PersonAvatar login={login} src={person.avatar_url} className="size-4 text-[8px]" />}
        {!login && <UserIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
        <span className="w-16 shrink-0 text-xs text-muted-foreground">{label}</span>
        <span className={cn("min-w-0 truncate text-xs", login ? "text-foreground" : "text-muted-foreground")}>
          {name || "No one"}
        </span>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-56 p-1.5">
        <PersonPickerList
          onSelect={(next) => {
            setOpen(false);
            setPerson.mutate({ id: ticket.id, role, login: next });
          }}
        />
      </PopoverContent>
    </Popover>
  );
};
