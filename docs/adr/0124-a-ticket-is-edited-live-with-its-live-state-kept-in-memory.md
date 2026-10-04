# A ticket is edited live, with its live state kept only in memory

A ticket's title and body autosaved through `PATCH /api/tickets/{id}` on a debounce (ADR 0050), and the save waited
for every ticket list to refetch before it cleared "saving…". A slow or failing list kept the indicator stuck, and
two people on one ticket overwrote each other, while the same people on a doc saw each other's typing.

Decision: a ticket is edited live the way a doc is. A third collab hub serves `/ws/collab/tickets/{id}`, keyed on
the ticket id, with the docs relay, protocol, and browser provider. Joining takes `tickets:write` to edit and
`tickets:read` to view, checked on the ticket's project. A commit writes the title and body through the tickets
use-case with `ticket.updated`; an empty title means unchanged, as for docs. A body written from outside, by
`ticket_update` or `PATCH /api/tickets/{id}`, runs through the room's reset (ADR 0109), so open editors reload it.

The room's state lives in memory (`collab.MemoryStore`), as a note's does (ADR 0110), because the docs update log
references `docs(id)` and a ticket needs no version history. The first editor to join an empty room seeds it from
the body.

The trade-off: a restart discards the last few seconds of typing in an open room, and a browser that reconnects
after one joins an empty room with its old state, the gap ADR 0109 names for a lost reset seq. In exchange there is
no new table and no migration on a database with live users.

Amends ADR 0050: a ticket no longer autosaves on a debounce; its meta row shows presence and Live instead of
Saving/Saved.

Decided 2026-10-04.
