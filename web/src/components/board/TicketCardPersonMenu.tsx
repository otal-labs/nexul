import { personLabel } from "@nexul/client-core/person";

import { NoPersonAvatar } from "@/components/NoPersonAvatar";
import { PersonAvatar } from "@/components/PersonAvatar";
import { ContextMenuRadioGroup, ContextMenuRadioItem } from "@/components/ui/context-menu";
import { useFetchProjectPeople } from "@/hooks/PeopleHooks";
import { useSetTicketPerson } from "@/hooks/TicketHooks";
import type { Ticket, TicketRole } from "@/models/Ticket";

interface TicketCardPersonMenuProps {
  ticket: Ticket;
  role: TicketRole;
  /** The login in the role now; empty is No one. */
  selected: string;
}

// The project's people, the same list the rail's picker offers; typing a name's first letters jumps to it.
export const TicketCardPersonMenu = ({ ticket, role, selected }: TicketCardPersonMenuProps) => {
  const { data: people } = useFetchProjectPeople(ticket.project_id);
  const setPerson = useSetTicketPerson();
  return (
    <ContextMenuRadioGroup value={selected} onValueChange={(login) => setPerson.mutate({ id: ticket.id, role, login })}>
      <ContextMenuRadioItem value="" indicator="end" textValue="No one">
        <NoPersonAvatar />
        <span className="text-muted-foreground">No one</span>
      </ContextMenuRadioItem>
      {people?.map((person) => (
        <ContextMenuRadioItem key={person.user_id} value={person.login} indicator="end" textValue={personLabel(person)}>
          <PersonAvatar login={person.login} name={personLabel(person)} src={person.avatar_url} />
          <span className="min-w-0 truncate">{personLabel(person)}</span>
        </ContextMenuRadioItem>
      ))}
    </ContextMenuRadioGroup>
  );
};
