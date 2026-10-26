import { unknownPerson, type Person } from "@nexul/client-core/person";

// A ticket names its developer and tester by user id beside the login, or by the login alone for someone who has not
// signed in yet (ADR 0170).
export interface TicketPeople {
  developer: string;
  developer_id?: string;
  tester: string;
  tester_id?: string;
}

// key is what matches a person, the user id or else the login; login names whoever the directory doesn't hold.
export interface TicketPersonRef {
  key: string;
  login: string;
}

export const ticketDeveloper = (ticket: TicketPeople): TicketPersonRef => ({
  key: ticket.developer_id || ticket.developer,
  login: ticket.developer,
});

export const ticketTester = (ticket: TicketPeople): TicketPersonRef => ({
  key: ticket.tester_id || ticket.tester,
  login: ticket.tester,
});

// Whether a ticket's person is this account; a server without ids names everyone by login.
export const namesAccount = (ref: TicketPersonRef, account: { id: string; login: string }): boolean =>
  ref.key !== "" && (ref.key === account.id || ref.key === account.login);

// The directory's entry for a ticket's person by id, else by login, else the login the ticket carries.
export const findTicketPerson = (people: Person[] | undefined, ref: TicketPersonRef): Person =>
  people?.find((p) => p.user_id === ref.key) ?? people?.find((p) => p.login === ref.login) ?? unknownPerson(ref.login);
