# 33 — Polish from the first live run

**What to build:** The ticket page follows a run's consequences live: a status change made by a play and a branch or PR linked through MCP reach the open page without a reload (invalidate the ticket, its links, and the board on `ticket.updated` and on the `play.run_finished` frame). The seeded instruction texts stop naming a `whoami` tool that does not exist and instead say to confirm access with `ticket_get` on the ticket (or `doc_get` on the doc). The run dialog's description no longer doubles the full stop when the play's description already ends with one. The ticket page's Thread section refreshes when a play opens the thread, so it shows the Started message instead of "Start chat" until reload. The trail row's failed summary truncates the harness error to one line with the full text in the detail.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] After a done run the ticket page shows the new column and the linked branch and PR without a reload
- [ ] Both seeded play texts (migration and Go seed) reference tools that exist
- [ ] Description punctuation is normalised in one place
- [ ] Starting a run invalidates the ticket thread query; the thread appears without a reload
- [ ] An extracted image reads `[image: <name>, attached to this turn]` in the prompt so the Agent connects the attachment with the memory or body it came from
- [ ] Failed trail rows stay one line
- [ ] A run stopped before the harness streamed anything ends `interrupted`, not `done`: the T3 terminal state after an interrupt is mapped from the stop, not from the harness's completion event
