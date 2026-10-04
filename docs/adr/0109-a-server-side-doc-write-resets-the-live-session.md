# A server-side doc write resets the live session, and one editor seeds an empty room

A doc open in the editor lives in two places: the canonical body on the doc, and the room's Y.js state (the
update log in `collab_updates`) that editors replay and merge. A body written outside the room, by `doc_update`, a
play, or `PUT /api/docs/{id}`, only reached the first. Open editors kept showing the room's state, later joiners
replayed it, and the next browser commit wrote it back over the new body, so the write was silently lost.

Decision: a write that changes a doc's body wins over the live session. The docs use-case runs the write through
the collab hub (`docs.LiveSessions`, wired in `server/cmd`), which holds the room's write lock across the write, then
drops the room's stored state, raises its seq past everything the room held, and sends every participant a `reset`
frame. Participants that joined before the reset are stale: the relay drops their updates and commits, and drops any
commit whose `base_seq` predates the reset. A browser answering `reset` reloads the doc and opens a fresh session. A
browser that was offline during the reset names the last seq it saw when it rejoins (`since`), and is answered with
`reset` instead of a replay it would merge its old state into. A title-only change does not reset the room.

An empty room is seeded once: the hub sends a `seed` frame to the first editor that joins it, and passes the seed to
another editor if that one leaves before seeding. Every other joiner waits for the seeded state, since two clients
loading the same body into the room would show it twice. Seeding is relayed like an edit but never committed, so
merely opening a doc writes nothing back.

The trade-off: whatever an editor typed between its last relayed update and the reset is discarded, because the
server write is the newer intent and merging two unrelated Y.js histories would duplicate the doc. The hub keeps the
last reset seq per room in memory, so a browser that is offline across both a reset and a server restart rejoins
with its old state; persisting the seq beside `collab_sessions` is the fix if that shows up.

The same reset is keyed on the room id, not on docs, so another live-edited body can reuse it.

Rooms written over before this decision never got their reset, so migration 0065 drops every room whose newest
update is no newer than its doc's last unnamed version row; the next editor seeds it from the body.

Decided 2026-10-02.
