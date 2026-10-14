# 16: Doors the web can close alone

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, `practices/native.md`, `practices/react-guide.md`, the spec.

## What to build

- Ticket delete.
- Delete for the DNS records the web made.
- An Archived filter and Restore for docs.
- On the phone: message edit and delete, and deploy cancel.
- The five copy fixes in the spec's table.
- Guide pages for each.

## Acceptance criteria

- [ ] Each way out is tested, and each confirm names what goes.
- [ ] Verified at 768, 1024 and 1440, and on the emulator for the phone.

## Answer

No server work was needed; every route and event already existed.

- **Ticket delete.** The ticket page header has a ⋯ with Delete, shown with `tickets:delete` (a new `deleteTickets`
  area in `client-core/permissions.ts`). The confirm names the ticket and says what goes with it and that it can't be
  undone. A ticket page returns to the project's board; one shown inside the Inbox stays put.
- **DNS records.** The records the web makes are the instance record (bare and proxy paths); the tunnel route's CNAME
  already goes with the tunnel (ticket 17). The Go live step of Set up DNS has Delete this record, hidden without
  `dns:delete`, with a confirm naming the record and where it pointed; afterwards the step goes back to the hostname.
- **Archived docs.** An archive toggle in the docs list header swaps the list to archived docs, which no longer sit
  among the live ones. Archive and Restore are in the row menu (`docs:write`), beside the ones on the doc page.
- **Phone.** Holding your own message opens a sheet with Edit (the text goes to the composer, Save or Cancel) and
  Delete (a confirm that says it is permanent for everyone); both are accessibility actions on the bubble. A waiting or
  running deploy has Cancel deploy for whoever may deploy the stack, with a confirm naming the stack.
- **Copy.** The five fixes in the spec's table, and the guide pages for ticket delete, archived docs, DNS record
  delete, the automations host, Reactivate and the phone app.
