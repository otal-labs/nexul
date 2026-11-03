import { useState, type SyntheticEvent } from "react";

import { personLabel, type Person } from "@nexul/client-core/person";
import type { TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { Microheader } from "@/components/Microheader";
import { NoPersonAvatar } from "@/components/NoPersonAvatar";
import { PersonAvatar } from "@/components/PersonAvatar";
import { PersonPickerList } from "@/components/ticket/PersonPickerList";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSetTicketPerson } from "@/hooks/TicketHooks";
import { TicketRole, type Ticket } from "@/models/Ticket";

interface CardPersonProps {
  ticket: Ticket;
  person: TicketPersonRef & { role: TicketRole };
  shown: Person;
  // False for a viewer who can't write the ticket, and on the drag overlay.
  assignable: boolean;
}

interface CardPersonPickerProps {
  ticket: Ticket;
  person: TicketPersonRef & { role: TicketRole };
  onPicked: () => void;
}

interface CardPersonFaceProps {
  login: string;
  shown: Person;
  label: string;
}

const roleLabel = (role: TicketRole) => (role === TicketRole.Tester ? "Tester" : "Developer");

// The card opens the ticket on click and drags on press; neither may fire from the picker or its portalled popover.
const keepFromCard = (event: SyntheticEvent) => event.stopPropagation();
const cardIsolation = { onClick: keepFromCard, onMouseDown: keepFromCard, onTouchStart: keepFromCard, onKeyDown: keepFromCard };

const CardPersonFace = ({ login, shown, label }: CardPersonFaceProps) => (
  <>
    {login && <PersonAvatar login={login} name={personLabel(shown)} src={shown.avatar_url} label={label} className="size-4" />}
    {!login && <NoPersonAvatar label={label} className="size-4" />}
  </>
);

// Mounted only while the popover is open, so a board of cards holds no edit per card.
const CardPersonPicker = ({ ticket, person, onPicked }: CardPersonPickerProps) => {
  const setPerson = useSetTicketPerson();
  return (
    <PersonPickerList
      projectId={ticket.project_id}
      selected={person.login}
      onSelect={(login) => {
        setPerson.mutate({ id: ticket.id, role: person.role, login });
        onPicked();
      }}
    />
  );
};

// Whoever acts next on the card's eyebrow; a click reassigns that role in place, the way the ticket rail does.
export const CardPerson = ({ ticket, person, shown, assignable }: CardPersonProps) => {
  const [open, setOpen] = useState(false);
  const role = roleLabel(person.role);
  const label = `${role}: ${person.login ? personLabel(shown) : "no one"}`;

  return (
    <>
      {!assignable && (
        <span role="img" aria-label={label} className="shrink-0">
          <CardPersonFace login={person.login} shown={shown} label={label} />
        </span>
      )}
      {assignable && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label={label}
              {...cardIsolation}
              className="relative shrink-0 cursor-pointer rounded-full transition-[box-shadow] duration-150 ease-standard after:absolute after:-inset-1 after:content-[''] hover:ring-2 hover:ring-muted-foreground/40 data-[state=open]:ring-2 data-[state=open]:ring-muted-foreground/40"
            >
              <CardPersonFace login={person.login} shown={shown} label={label} />
            </button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-56 p-1.5" {...cardIsolation}>
            <Microheader className="px-1 pb-1.5">{role}</Microheader>
            <CardPersonPicker ticket={ticket} person={person} onPicked={() => setOpen(false)} />
          </PopoverContent>
        </Popover>
      )}
    </>
  );
};
