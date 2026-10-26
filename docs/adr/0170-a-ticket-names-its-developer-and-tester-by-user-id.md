# A ticket names its developer and tester by user id

A ticket kept its developer and tester as login strings, and every client compared them exactly, so one person could
split in two: a login saved in another case, a renamed login, or a second sign-in provider.

Decision: `tickets.developer` and `tickets.tester` hold user ids, and every read joins `users` to return the login
beside the id (`developer` and `developer_id`, `tester` and `tester_id`), on the HTTP gateway, the MCP tools and every
event that carries a ticket. Clients and consumers match people by id.

- **Logins still come in.** `ticket_create`, `ticket_update` and the developer and tester routes take a login and
  resolve it to the account, as before; the change events gain `from_id` and `to_id` beside `from` and `to`.
- **A login nobody has signed in with is a placeholder.** It is kept as typed, reads back with an empty id, and is
  claimed by the first account created with that login, in the same transaction. Migration 0137 converts every login
  an account holds, preferring the exact spelling when accounts differ only in case.

Rejected: clearing unknown logins in the migration and refusing them afterwards, which loses assignments on
instances already running and refuses what the routes accept today; a separate id column beside the login, which
keeps two sources of truth that drift on a rename.

Decided 2026-10-26.
