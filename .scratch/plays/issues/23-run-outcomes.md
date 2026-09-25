# 23 — Run outcomes: move-to, events, notifications, timeout, Stop

**What to build:** When a trail ends, the right things happen. On `done` for a ticket play, the ticket moves to the column chosen at run time unless that would move it to an earlier stage than it now sits in, or the column no longer exists, in which case the move is skipped with a system note in the thread; the status change carries actor kind `play` with the play's name and the trail id, and the `:mcp` suffix when started through MCP. On any terminal outcome the starter gets an inbox notification; on failure or interrupt a system note says why and any partial reply is posted. `play.run_started` and `play.run_finished` (with `outcome`) are catalog rows written through the outbox, and a `play.run` live topic pushes trail state changes. A trail with no harness update for the silence window (about fifteen minutes, one instance-wide setting) ends `failed`. `trail_update` and its route interrupt a running trail for the starter or anyone with `plays:write`, keeping the record.

**Blocked by:** 22

**Status:** done

- [ ] A done run moves the ticket to In review; a done run on a ticket already in Done (finished by merge mid-run) skips with a note; a deleted column skips with a note
- [ ] The status-changed event and the ticket's history show the `play` actor with name and trail id
- [ ] Both catalog rows exist with schemas; an automation subscribed to `play.run_finished` receives the outcome
- [ ] Silence for the window ends the trail `failed` with a note; a run that keeps streaming past the window continues
- [ ] Stop from the starter and from a `plays:write` holder interrupts; a third user is refused; the trail keeps its activity lines
- [ ] The starter's inbox shows one notification per terminal outcome
